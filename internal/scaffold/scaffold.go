// Package scaffold generates or completes a .env file from a parsed example.
package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/moveeeax/envsync/internal/dotenv"
)

// Plan describes what Apply would change.
type Plan struct {
	Created bool
	Added   []string
}

// Render returns the file body for the missing keys, one KEY=value line each.
// Existing keys are preserved by the caller; only missing lines are produced.
func Render(example *dotenv.File, existing *dotenv.File) (string, []string) {
	var b strings.Builder
	var added []string
	for _, v := range example.Vars {
		if _, ok := existing.Get(v.Key); ok {
			continue
		}
		fmt.Fprintf(&b, "%s=%s\n", v.Key, v.Value)
		added = append(added, v.Key)
	}
	return b.String(), added
}

// Apply writes missing keys from example into the .env at path, creating it if
// absent. It never overwrites existing keys.
func Apply(example *dotenv.File, path string) (Plan, error) {
	existing := &dotenv.File{}
	created := false
	if f, err := dotenv.ParseFile(path); err == nil {
		existing = f
	} else if os.IsNotExist(err) {
		created = true
	} else {
		return Plan{}, err
	}

	body, added := Render(example, existing)
	if len(added) == 0 {
		return Plan{Created: false, Added: nil}, nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Plan{}, err
	}
	defer f.Close()

	if !created {
		if info, err := f.Stat(); err == nil && info.Size() > 0 {
			if _, err := f.WriteString("\n"); err != nil {
				return Plan{}, err
			}
		}
	}
	if _, err := f.WriteString(body); err != nil {
		return Plan{}, err
	}
	return Plan{Created: created, Added: added}, nil
}
