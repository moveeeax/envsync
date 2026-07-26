package dotenv

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	in := `# comment
export PORT=8080
API_URL="https://example.com"
EMPTY=
QUOTED='hello world'
`
	f, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"PORT":    "8080",
		"API_URL": "https://example.com",
		"EMPTY":   "",
		"QUOTED":  "hello world",
	}
	for k, v := range want {
		got, ok := f.Get(k)
		if !ok {
			t.Fatalf("missing key %s", k)
		}
		if got.Value != v {
			t.Errorf("%s = %q, want %q", k, got.Value, v)
		}
	}
	if keys := f.Keys(); !reflect.DeepEqual(keys, []string{"PORT", "API_URL", "EMPTY", "QUOTED"}) {
		t.Errorf("key order = %v", keys)
	}
}

func TestParseAnnotationsPrecedingLine(t *testing.T) {
	in := `# @type=int @required
PORT=8080
# @type=enum(dev,staging,prod) @required
ENV=dev
# @type=url
API_URL=https://example.com
# @type=bool
DEBUG=true
`
	f, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		key  string
		spec Spec
	}{
		{"PORT", Spec{Type: "int", Required: true}},
		{"ENV", Spec{Type: "enum", Required: true, Enum: []string{"dev", "staging", "prod"}}},
		{"API_URL", Spec{Type: "url"}},
		{"DEBUG", Spec{Type: "bool"}},
	}
	for _, c := range cases {
		got, ok := f.Get(c.key)
		if !ok {
			t.Fatalf("missing %s", c.key)
		}
		if !reflect.DeepEqual(got.Spec, c.spec) {
			t.Errorf("%s spec = %+v, want %+v", c.key, got.Spec, c.spec)
		}
	}
}

func TestParseInlineAnnotation(t *testing.T) {
	f, err := Parse(strings.NewReader(`TIMEOUT=30 # @type=int @required`))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.Get("TIMEOUT")
	if got.Value != "30" {
		t.Errorf("value = %q, want 30", got.Value)
	}
	if got.Spec.Type != "int" || !got.Spec.Required {
		t.Errorf("spec = %+v", got.Spec)
	}
}

func TestInlineHashInsideQuotesIsLiteral(t *testing.T) {
	f, err := Parse(strings.NewReader(`COLOR="#ffffff"`))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.Get("COLOR")
	if got.Value != "#ffffff" {
		t.Errorf("value = %q, want #ffffff", got.Value)
	}
}

// "@enum=a, b, c" is as natural to write as "@enum=a,b,c"; dropping every
// member after the first silently turns a valid .env into a mismatch.
func TestEnumAnnotationToleratesSpacesAfterCommas(t *testing.T) {
	cases := map[string]string{
		"spaced":            "# @enum=dev, staging, prod\nAPP_ENV=dev\n",
		"tight":             "# @enum=dev,staging,prod\nAPP_ENV=dev\n",
		"spaced_then_flag":  "# @enum=dev, staging, prod @required\nAPP_ENV=dev\n",
		"parenthesised":     "# @type=enum(dev, staging, prod)\nAPP_ENV=dev\n",
		"trailing_prose":    "# @enum=dev,staging,prod pick one\nAPP_ENV=dev\n",
		"tight_then_flag":   "# @enum=dev,staging,prod @required\nAPP_ENV=dev\n",
		"inline_annotation": "APP_ENV=dev # @enum=dev, staging, prod\n",
	}
	want := []string{"dev", "staging", "prod"}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			got, ok := f.Get("APP_ENV")
			if !ok {
				t.Fatal("missing APP_ENV")
			}
			if got.Spec.Type != "enum" {
				t.Errorf("type = %q, want enum", got.Spec.Type)
			}
			if !reflect.DeepEqual(got.Spec.Enum, want) {
				t.Errorf("enum = %v, want %v", got.Spec.Enum, want)
			}
		})
	}
}

func TestEnumAnnotationSetsRequired(t *testing.T) {
	f, err := Parse(strings.NewReader("# @enum=dev, prod @required\nAPP_ENV=dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.Get("APP_ENV")
	if !got.Spec.Required {
		t.Errorf("@required after a spaced @enum list was swallowed: %+v", got.Spec)
	}
}

// A '#' immediately after '=' has no preceding whitespace, so per shell
// word-splitting rules it is part of the value, not a comment marker.
// "COLOR=#336699" (a hex color, or a git-SHA-like token) must parse to the
// value "#336699", not silently become empty with the whole thing swallowed
// as a dropped comment.
func TestHashAtStartOfValueIsLiteralNotComment(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"hex_color":      {"THEME_COLOR=#336699", "#336699"},
		"just_hash":      {"KEY=#", "#"},
		"hash_then_text": {"KEY=#deadbeef and more", "#deadbeef and more"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(c.in + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			got, ok := f.Get("KEY")
			if name == "hex_color" {
				got, ok = f.Get("THEME_COLOR")
			}
			if !ok {
				t.Fatal("key not found")
			}
			if got.Value != c.want {
				t.Errorf("value = %q, want %q", got.Value, c.want)
			}
		})
	}
}

// A '#' preceded by whitespace is still a comment, whether or not the value
// before it happens to start with '#' too.
func TestSpaceThenHashIsStillAComment(t *testing.T) {
	f, err := Parse(strings.NewReader("KEY=value #not part of the value\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.Get("KEY")
	if got.Value != "value" {
		t.Errorf("value = %q, want %q", got.Value, "value")
	}
}

func TestUntypedDefaultsToString(t *testing.T) {
	f, _ := Parse(strings.NewReader("NAME=bob"))
	got, _ := f.Get("NAME")
	if got.Spec.Type != "string" {
		t.Errorf("type = %q, want string", got.Spec.Type)
	}
}
