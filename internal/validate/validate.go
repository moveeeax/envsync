// Package validate compares a .env file against a schema-annotated
// .env.example and reports missing, extra and type-mismatched variables.
package validate

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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

// Options tunes validation severity.
type Options struct {
	// Strict makes extra variables count as failures.
	Strict bool
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
		if msg, ok := checkType(got.Value, spec.Spec); !ok {
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

// checkType validates a raw value against its spec. Empty values are only
// rejected when the variable is required.
func checkType(value string, spec dotenv.Spec) (string, bool) {
	if value == "" {
		if spec.Required {
			return "required variable is empty", false
		}
		return "", true
	}
	switch spec.Type {
	case "", "string":
		return "", true
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Sprintf("expected int, got %q", value), false
		}
	case "bool":
		if !isBool(value) {
			return fmt.Sprintf("expected bool, got %q", value), false
		}
	case "url":
		if !isURL(value) {
			return fmt.Sprintf("expected url, got %q", value), false
		}
	case "enum":
		if !inList(value, spec.Enum) {
			return fmt.Sprintf("expected one of [%s], got %q", strings.Join(spec.Enum, ", "), value), false
		}
	default:
		return "", true // unknown types are treated as free-form strings
	}
	return "", true
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
