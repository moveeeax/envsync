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

func TestUntypedDefaultsToString(t *testing.T) {
	f, _ := Parse(strings.NewReader("NAME=bob"))
	got, _ := f.Get("NAME")
	if got.Spec.Type != "string" {
		t.Errorf("type = %q, want string", got.Spec.Type)
	}
}
