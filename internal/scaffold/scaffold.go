// Package scaffold generates or completes a .env file from a parsed example.
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moveeeax/envsync/internal/dotenv"
)

// FileMode is the permission a .env is written with. A .env is a credential
// file, so it must not be group- or world-readable.
const FileMode os.FileMode = 0o600

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
		fmt.Fprintf(&b, "%s=%s\n", v.Key, quoteIfNeeded(v.Value))
		added = append(added, v.Key)
	}
	return b.String(), added
}

// quoteIfNeeded wraps v in quotes when writing it out unquoted would not
// round-trip back through dotenv.Parse to the same value: leading/trailing
// whitespace is trimmed by the parser unless quoted, and a " #" (or "\t#")
// in the middle of an unquoted value is parsed as the start of a trailing
// comment, truncating everything after it. Both are real shapes for a
// .env.example default, e.g. `TITLE="My App #1"`. Values that need quoting
// but contain a '"' fall back to single quotes; if they contain both quote
// characters this minimal writer has no escape syntax and best-effort wraps
// in double quotes anyway, since leaving it unquoted is guaranteed to corrupt
// it.
func quoteIfNeeded(v string) string {
	if !needsQuoting(v) {
		return v
	}
	if !strings.Contains(v, `"`) {
		return `"` + v + `"`
	}
	if !strings.Contains(v, "'") {
		return "'" + v + "'"
	}
	return `"` + v + `"`
}

func needsQuoting(v string) bool {
	if strings.TrimSpace(v) != v {
		return true
	}
	return strings.Contains(v, " #") || strings.Contains(v, "\t#")
}

// Apply writes missing keys from example into the .env at path, creating it if
// absent. It never overwrites existing keys.
//
// The new content is staged in a temporary file in the same directory and moved
// into place with a rename, so a failure part-way through cannot leave the
// caller's .env truncated or half-written. The rename also resets the mode to
// FileMode, tightening a .env that was previously left world-readable.
func Apply(example *dotenv.File, path string) (Plan, error) {
	existing := &dotenv.File{}
	created := false

	current, err := os.ReadFile(path)
	switch {
	case err == nil:
		f, perr := dotenv.Parse(bytes.NewReader(current))
		if perr != nil {
			return Plan{}, perr
		}
		existing = f
	case os.IsNotExist(err):
		created = true
		current = nil
	default:
		return Plan{}, err
	}

	body, added := Render(example, existing)
	if len(added) == 0 {
		return Plan{Created: false, Added: nil}, nil
	}

	var buf bytes.Buffer
	buf.Write(current)
	if len(current) > 0 {
		if !bytes.HasSuffix(current, []byte("\n")) {
			buf.WriteByte('\n') // do not glue our first key onto a partial line
		}
		buf.WriteByte('\n') // blank line between the old content and ours
	}
	buf.WriteString(body)

	if err := writeAtomic(path, buf.Bytes()); err != nil {
		return Plan{}, err
	}
	return Plan{Created: created, Added: added}, nil
}

// writeAtomic writes data to path via a temporary file in the same directory
// followed by a rename, so path is either the old content or the new content
// and never a truncated mixture of the two.
func writeAtomic(path string, data []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".envsync-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(tmp)
		}
	}()

	if err = f.Chmod(FileMode); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	err = os.Rename(tmp, path)
	return err
}
