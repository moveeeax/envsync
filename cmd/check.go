package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/moveeeax/envsync/internal/dotenv"
	"github.com/moveeeax/envsync/internal/scaffold"
	"github.com/moveeeax/envsync/internal/validate"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var (
		examplePath string
		envPath     string
		asJSON      bool
		strict      bool
		fix         bool
		showValues  bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate .env against .env.example",
		RunE: func(cmd *cobra.Command, args []string) error {
			example, err := dotenv.ParseFile(examplePath)
			if err != nil {
				return fmt.Errorf("read example %s: %w", examplePath, err)
			}

			if fix {
				plan, err := scaffold.Apply(example, envPath)
				if err != nil {
					return fmt.Errorf("fix %s: %w", envPath, err)
				}
				reportFix(cmd, plan, envPath, asJSON)
			}

			env, err := dotenv.ParseFile(envPath)
			if err != nil {
				if os.IsNotExist(err) {
					env = &dotenv.File{}
				} else {
					return fmt.Errorf("read env %s: %w", envPath, err)
				}
			}

			res := validate.Validate(example, env, validate.Options{Strict: strict, ShowValues: showValues})
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					return err
				}
			} else {
				printHuman(cmd, res)
			}
			if !res.OK {
				return errFail
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&examplePath, "example", "e", ".env.example", "path to the schema-annotated example file")
	cmd.Flags().StringVarP(&envPath, "env", "f", ".env", "path to the .env file to validate")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat extra variables as failures")
	cmd.Flags().BoolVar(&fix, "fix", false, "scaffold missing keys into the .env before validating")
	cmd.Flags().BoolVar(&showValues, "show-values", false, "include the offending value in mismatch messages (may print secrets)")
	return cmd
}

// errFail is returned to signal a non-zero exit without printing a usage error.
var errFail = fmt.Errorf("validation failed")

// ErrValidationFailed exposes the sentinel returned when validation fails, so
// callers can distinguish an expected non-zero exit from a real error.
func ErrValidationFailed() error { return errFail }

func printHuman(cmd *cobra.Command, res validate.Result) {
	out := cmd.OutOrStdout()
	if len(res.Issues) == 0 {
		fmt.Fprintln(out, "OK: .env matches .env.example")
		return
	}
	for _, is := range res.Issues {
		fmt.Fprintf(out, "%-9s %s: %s\n", is.Kind, is.Key, is.Message)
	}
	fmt.Fprintf(out, "\n%d issue(s) found\n", len(res.Issues))
}

func reportFix(cmd *cobra.Command, plan scaffold.Plan, path string, asJSON bool) {
	if asJSON {
		return
	}
	out := cmd.OutOrStdout()
	switch {
	case len(plan.Added) == 0:
		fmt.Fprintf(out, "fix: %s already complete, nothing added\n", path)
	case plan.Created:
		fmt.Fprintf(out, "fix: created %s with %d variable(s)\n", path, len(plan.Added))
	default:
		fmt.Fprintf(out, "fix: added %d variable(s) to %s\n", len(plan.Added), path)
	}
}
