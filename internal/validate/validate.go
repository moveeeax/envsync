// Package validate compares a .env file against a schema-annotated
// .env.example and reports missing, extra and type-mismatched variables.
package validate

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/moveeeax/envsync/internal/dotenv"
)

// Kind categorises an Issue.
type Kind string

const (
	Missing  Kind = "missing"  // required var absent from .env
	Extra    Kind = "extra"    // var present in .env but not in .env.example
	Mismatch Kind = "mismatch" // value fails its declared type/enum
)

// Issue is a single validation finding.
type Issue struct {
	Key     string `json:"key"`
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
}

// Options tunes validation severity and output.
type Options struct {
	// Strict makes extra variables count as failures.
	Strict bool
	// ShowValues includes the offending value in mismatch messages. It is off
	// by default because those messages are printed to stdout and land in CI
	// logs, and a mistyped variable is very often a secret.
	ShowValues bool
}

// Result holds all findings plus a pass/fail verdict.
type Result struct {
	Issues []Issue `json:"issues"`
	OK     bool    `json:"ok"`
}

// Validate checks env against the example schema.
func Validate(example, env *dotenv.File, opts Options) Result {
	var issues []Issue

	for _, spec := range example.Vars {
		got, present := env.Get(spec.Key)
		if !present {
			if spec.Spec.Required {
				issues = append(issues, Issue{spec.Key, Missing, "required variable is missing"})
			}
			continue
		}
		// An empty value for a required variable is reported as Missing, not
		// Mismatch: it is the same failure as the variable not being there at
		// all, and consumers filter on kind.
		if got.Value == "" {
			if spec.Spec.Required {
				issues = append(issues, Issue{spec.Key, Missing, "required variable is present but empty"})
			}
			continue
		}
		if msg, ok := checkType(got.Value, spec.Spec, opts.ShowValues); !ok {
			issues = append(issues, Issue{spec.Key, Mismatch, msg})
		}
	}

	for _, v := range env.Vars {
		if _, known := example.Get(v.Key); !known {
			issues = append(issues, Issue{v.Key, Extra, "variable is not declared in the example"})
		}
	}

	ok := true
	for _, is := range issues {
		switch is.Kind {
		case Missing, Mismatch:
			ok = false
		case Extra:
			if opts.Strict {
				ok = false
			}
		}
	}
	return Result{Issues: issues, OK: ok}
}

// checkType validates a raw value against its spec. Empty values always pass
// here; the required-but-empty case is handled by Validate.
func checkType(value string, spec dotenv.Spec, showValues bool) (string, bool) {
	if value == "" {
		return "", true
	}
	got := describe(value, showValues)
	switch spec.Type {
	case "", "string":
		return "", true
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Sprintf("expected int, got %s", got), false
		}
	case "bool":
		if !isBool(value) {
			return fmt.Sprintf("expected bool, got %s", got), false
		}
	case "url":
		if !isURL(value) {
			return fmt.Sprintf("expected url, got %s", got), false
		}
	case "enum":
		// The allowed list comes from the committed example file, so it is
		// safe to print; the actual value is not.
		if !inList(value, spec.Enum) {
			return fmt.Sprintf("expected one of [%s], got %s", strings.Join(spec.Enum, ", "), got), false
		}
	default:
		return "", true // unknown types are treated as free-form strings
	}
	return "", true
}

// describe renders the offending value for an error message. Unless the caller
// explicitly asked for values, only the length is disclosed — the value may be
// a credential and these messages go to stdout and CI logs.
func describe(value string, showValues bool) string {
	if showValues {
		return strconv.Quote(value)
	}
	return fmt.Sprintf("a %d-character value", utf8.RuneCountInString(value))
}

func isBool(v string) bool {
	switch strings.ToLower(v) {
	case "true", "false", "1", "0", "yes", "no", "on", "off":
		return true
	}
	return false
}

func isURL(v string) bool {
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
