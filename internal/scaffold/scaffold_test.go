package scaffold

import (
	"os"
	"path/filepath"
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

func TestApplyCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	ex := parse(t, "PORT=8080\nENV=dev\n")

	plan, err := Apply(ex, path)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Created || len(plan.Added) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "PORT=8080") || !strings.Contains(string(body), "ENV=dev") {
		t.Errorf("body = %q", body)
	}
}

func TestApplyAppendsOnlyMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("PORT=3000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ex := parse(t, "PORT=8080\nENV=dev\n")

	plan, err := Apply(ex, path)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Created {
		t.Error("should not report created for existing file")
	}
	if len(plan.Added) != 1 || plan.Added[0] != "ENV" {
		t.Fatalf("added = %v", plan.Added)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "PORT=3000") {
		t.Error("must not overwrite existing PORT")
	}
	if strings.Contains(string(body), "PORT=8080") {
		t.Error("must not append duplicate PORT")
	}
	if !strings.Contains(string(body), "ENV=dev") {
		t.Error("must append missing ENV")
	}
}

func TestApplyIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	ex := parse(t, "A=1\nB=2\n")

	if _, err := Apply(ex, path); err != nil {
		t.Fatal(err)
	}
	plan, err := Apply(ex, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Added) != 0 {
		t.Errorf("second apply added %v, want none", plan.Added)
	}
}
