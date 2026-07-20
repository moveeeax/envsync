package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

const exampleFile = `# @type=int @required
PORT=8080
# @type=enum(dev,prod) @required
ENV=dev
`

func TestCheckPasses(t *testing.T) {
	dir := t.TempDir()
	ex := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, ex, exampleFile)
	write(t, env, "PORT=3000\nENV=prod\n")

	out, err := run(t, "check", "-e", ex, "-f", env)
	if err != nil {
		t.Fatalf("unexpected err: %v (%s)", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("out = %q", out)
	}
}

func TestCheckFailsNonZero(t *testing.T) {
	dir := t.TempDir()
	ex := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, ex, exampleFile)
	write(t, env, "ENV=staging\n") // PORT missing + ENV bad enum

	_, err := run(t, "check", "-e", ex, "-f", env)
	if !errors.Is(err, ErrValidationFailed()) {
		t.Fatalf("expected validation failure, got %v", err)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := t.TempDir()
	ex := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, ex, exampleFile)
	write(t, env, "PORT=x\nENV=dev\n")

	out, err := run(t, "check", "-e", ex, "-f", env, "--json")
	if !errors.Is(err, ErrValidationFailed()) {
		t.Fatalf("expected failure, got %v", err)
	}
	var res struct {
		OK     bool `json:"ok"`
		Issues []struct {
			Key, Kind string
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if res.OK {
		t.Error("ok should be false")
	}
	if len(res.Issues) == 0 {
		t.Error("expected issues in json")
	}
}

func TestCheckFix(t *testing.T) {
	dir := t.TempDir()
	ex := filepath.Join(dir, ".env.example")
	env := filepath.Join(dir, ".env")
	write(t, ex, exampleFile)

	out, err := run(t, "check", "-e", ex, "-f", env, "--fix")
	if err != nil {
		t.Fatalf("fix+check should pass: %v (%s)", err, out)
	}
	body, _ := os.ReadFile(env)
	if !strings.Contains(string(body), "PORT=8080") || !strings.Contains(string(body), "ENV=dev") {
		t.Errorf("scaffolded .env = %q", body)
	}
}
