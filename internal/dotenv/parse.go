// Package dotenv parses .env files and the annotation comments used by
// .env.example schema files.
package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Spec is the schema declared for a variable via annotation comments in an
// .env.example file. A zero Spec means "string, optional".
type Spec struct {
	Type     string   // "string", "int", "bool", "url", "enum"
	Required bool     // true when @required is present
	Enum     []string // allowed values when Type == "enum"
}

// Var is a single parsed entry. For .env files Spec is zero; for .env.example
// files Spec carries the annotations that precede or trail the assignment.
type Var struct {
	Key   string
	Value string
	Line  int
	Spec  Spec
}

// File is the ordered result of parsing a dotenv file.
type File struct {
	Vars  []Var
	index map[string]int
}

// Get returns the variable for key and whether it exists.
func (f *File) Get(key string) (Var, bool) {
	i, ok := f.index[key]
	if !ok {
		return Var{}, false
	}
	return f.Vars[i], true
}

// Keys returns the keys in file order.
func (f *File) Keys() []string {
	keys := make([]string, len(f.Vars))
	for i, v := range f.Vars {
		keys[i] = v.Key
	}
	return keys
}

var (
	assignRe = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$`)
	typeRe   = regexp.MustCompile(`@type=([A-Za-z]+)(?:\(([^)]*)\))?`)
	// Members may be separated by ", " as well as ",". Without the optional
	// whitespace, "@enum=dev, staging, prod" captured only "dev," and every
	// other value was silently rejected as out of range.
	enumRe = regexp.MustCompile(`@enum=([^\s,]+(?:\s*,\s*[^\s,]+)*)`)
)

// ParseFile reads and parses the file at path.
func ParseFile(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse parses dotenv content from r.
func Parse(r io.Reader) (*File, error) {
	out := &File{index: map[string]int{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var pending Spec // annotations accumulated from standalone comment lines
	lineNo := 0

	for sc.Scan() {
		lineNo++
		raw := sc.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			pending = Spec{}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if spec, ok := parseAnnotations(trimmed); ok {
				pending = merge(pending, spec)
			}
			continue
		}

		m := assignRe.FindStringSubmatch(raw)
		if m == nil {
			pending = Spec{}
			continue
		}
		key := m[1]
		value, inlineComment := splitInline(m[2])

		spec := pending
		if inlineComment != "" {
			if s, ok := parseAnnotations(inlineComment); ok {
				spec = merge(spec, s)
			}
		}
		pending = Spec{}

		v := Var{Key: key, Value: unquote(value), Line: lineNo, Spec: normalize(spec)}
		if i, exists := out.index[key]; exists {
			out.Vars[i] = v // last assignment wins, mirroring dotenv semantics
		} else {
			out.index[key] = len(out.Vars)
			out.Vars = append(out.Vars, v)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// splitInline separates a value from a trailing " # ..." comment. A '#' inside
// quotes is treated as literal.
func splitInline(s string) (value, comment string) {
	inSingle, inDouble := false, false
	for i, r := range s {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
			}
		}
	}
	return strings.TrimSpace(s), ""
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseAnnotations extracts @type / @enum / @required from a comment string.
func parseAnnotations(comment string) (Spec, bool) {
	if !strings.Contains(comment, "@") {
		return Spec{}, false
	}
	var spec Spec
	found := false
	if m := typeRe.FindStringSubmatch(comment); m != nil {
		spec.Type = strings.ToLower(m[1])
		if m[2] != "" {
			spec.Enum = splitList(m[2])
		}
		found = true
	}
	if m := enumRe.FindStringSubmatch(comment); m != nil {
		spec.Enum = splitList(m[1])
		if spec.Type == "" {
			spec.Type = "enum"
		}
		found = true
	}
	if strings.Contains(comment, "@required") {
		spec.Required = true
		found = true
	}
	return spec, found
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func merge(base, over Spec) Spec {
	if over.Type != "" {
		base.Type = over.Type
	}
	if len(over.Enum) > 0 {
		base.Enum = over.Enum
	}
	if over.Required {
		base.Required = true
	}
	return base
}

func normalize(s Spec) Spec {
	if len(s.Enum) > 0 && s.Type == "" {
		s.Type = "enum"
	}
	if s.Type == "" {
		s.Type = "string"
	}
	return s
}

// String renders a Spec for human output.
func (s Spec) String() string {
	parts := []string{s.Type}
	if s.Type == "enum" && len(s.Enum) > 0 {
		parts[0] = fmt.Sprintf("enum(%s)", strings.Join(s.Enum, ","))
	}
	if s.Required {
		parts = append(parts, "required")
	}
	return strings.Join(parts, ", ")
}
