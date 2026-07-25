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

// A required variable that is present but empty is the same failure as one
// that is absent, and must be reported under the same kind so that consumers
// filtering on kind=="missing" actually see it.
func TestRequiredEmptyIsMissingNotMismatch(t *testing.T) {
	env := parse(t, "PORT=\nENV=dev\n")
	res := Validate(parse(t, example), env, Options{})
	if res.OK {
		t.Fatal("expected failure")
	}
	if !has(res.Issues, "PORT", Missing) {
		t.Errorf("empty required PORT should be reported as missing: %+v", res.Issues)
	}
	if has(res.Issues, "PORT", Mismatch) {
		t.Errorf("empty required PORT should not be reported as mismatch: %+v", res.Issues)
	}
}

func TestBoolVariants(t *testing.T) {
	for _, v := range []string{"true", "false", "1", "0", "yes", "no", "on", "off"} {
		if msg, ok := checkType(v, dotenv.Spec{Type: "bool"}, false); !ok {
			t.Errorf("%q should be valid bool: %s", v, msg)
		}
	}
	if _, ok := checkType("nope", dotenv.Spec{Type: "bool"}, false); ok {
		t.Error("nope should be invalid bool")
	}
}

// Mismatch messages are printed to stdout and end up in CI logs, so they must
// not echo the value: a mistyped variable is very often a credential.
func TestMismatchMessagesDoNotLeakValues(t *testing.T) {
	const secret = "not-a-real-token-abcdefghi"
	specs := []dotenv.Spec{
		{Type: "int"},
		{Type: "bool"},
		{Type: "url"},
		{Type: "enum", Enum: []string{"dev", "prod"}},
	}
	for _, spec := range specs {
		msg, ok := checkType(secret, spec, false)
		if ok {
			t.Fatalf("%s should reject %q", spec.Type, secret)
		}
		if strings.Contains(msg, secret) {
			t.Errorf("%s message leaks the value: %q", spec.Type, msg)
		}
		if !strings.Contains(msg, "26-character") {
			t.Errorf("%s message should describe the value length, got %q", spec.Type, msg)
		}
	}
}

func TestValidateDoesNotLeakValuesEndToEnd(t *testing.T) {
	const secret = "not-a-real-token-abcdefghi"
	env := parse(t, "PORT="+secret+"\nENV="+secret+"\n")
	res := Validate(parse(t, example), env, Options{})
	if res.OK {
		t.Fatal("expected failure")
	}
	for _, is := range res.Issues {
		if strings.Contains(is.Message, secret) {
			t.Errorf("issue %+v leaks the value", is)
		}
	}
}

func TestShowValuesOptsIntoEchoingTheValue(t *testing.T) {
	env := parse(t, "PORT=notanint\nENV=dev\n")
	res := Validate(parse(t, example), env, Options{ShowValues: true})
	found := false
	for _, is := range res.Issues {
		if is.Key == "PORT" && strings.Contains(is.Message, `"notanint"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("--show-values should echo the value: %+v", res.Issues)
	}
}
