package validate

import (
	"strings"
	"testing"

	"github.com/moveeeax/envsync/internal/dotenv"
)

func parse(t *testing.T, s string) *dotenv.File {
	t.Helper()
	f, err := dotenv.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

const example = `# @type=int @required
PORT=8080
# @type=enum(dev,prod) @required
ENV=dev
# @type=url
API_URL=https://example.com
# @type=bool
DEBUG=false
OPTIONAL=hi
`

func has(issues []Issue, key string, kind Kind) bool {
	for _, is := range issues {
		if is.Key == key && is.Kind == kind {
			return true
		}
	}
	return false
}

func TestValidClean(t *testing.T) {
	env := parse(t, "PORT=3000\nENV=prod\nAPI_URL=https://api.test\nDEBUG=true\nOPTIONAL=x\n")
	res := Validate(parse(t, example), env, Options{})
	if !res.OK || len(res.Issues) != 0 {
		t.Fatalf("expected clean pass, got %+v", res)
	}
}

func TestMissingRequired(t *testing.T) {
	env := parse(t, "ENV=dev\nAPI_URL=https://api.test\n")
	res := Validate(parse(t, example), env, Options{})
	if res.OK {
		t.Error("expected failure")
	}
	if !has(res.Issues, "PORT", Missing) {
		t.Errorf("PORT missing not reported: %+v", res.Issues)
	}
}

func TestTypeMismatches(t *testing.T) {
	env := parse(t, "PORT=notanint\nENV=staging\nAPI_URL=not a url\nDEBUG=maybe\n")
	res := Validate(parse(t, example), env, Options{})
	if res.OK {
		t.Error("expected failure")
	}
	for _, k := range []string{"PORT", "ENV", "API_URL", "DEBUG"} {
		if !has(res.Issues, k, Mismatch) {
			t.Errorf("%s mismatch not reported", k)
		}
	}
}

func TestExtraNonStrict(t *testing.T) {
	env := parse(t, "PORT=1\nENV=dev\nSURPRISE=1\n")
	res := Validate(parse(t, example), env, Options{})
	if !has(res.Issues, "SURPRISE", Extra) {
		t.Error("extra not reported")
	}
	if !res.OK {
		t.Error("extra should not fail without --strict")
	}
}

func TestExtraStrictFails(t *testing.T) {
	env := parse(t, "PORT=1\nENV=dev\nSURPRISE=1\n")
	res := Validate(parse(t, example), env, Options{Strict: true})
	if res.OK {
		t.Error("extra should fail with --strict")
	}
}

func TestOptionalEmptyOK(t *testing.T) {
	env := parse(t, "PORT=1\nENV=dev\nOPTIONAL=\n")
	res := Validate(parse(t, example), env, Options{})
	if !res.OK {
		t.Errorf("optional empty should pass: %+v", res.Issues)
	}
}

func TestBoolVariants(t *testing.T) {
	for _, v := range []string{"true", "false", "1", "0", "yes", "no", "on", "off"} {
		if msg, ok := checkType(v, dotenv.Spec{Type: "bool"}); !ok {
			t.Errorf("%q should be valid bool: %s", v, msg)
		}
	}
	if _, ok := checkType("nope", dotenv.Spec{Type: "bool"}); ok {
		t.Error("nope should be invalid bool")
	}
}
