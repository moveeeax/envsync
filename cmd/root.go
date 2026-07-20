package cmd

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// NewRootCmd builds the root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "envsync",
		Short:         "Validate a .env against a schema-annotated .env.example",
		Long:          "envsync checks a project's .env against a typed .env.example, reporting missing, extra and type-mismatched variables.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newCheckCmd())
	return root
}
