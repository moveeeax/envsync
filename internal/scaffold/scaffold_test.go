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

// A .env holds credentials. Whether we create it or extend one somebody else
// created, it must not be left group- or world-readable.
func TestApplyResultIsOwnerOnlyReadable(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")
		if _, err := Apply(parse(t, "TOKEN=changeme\n"), path); err != nil {
			t.Fatal(err)
		}
		assertMode(t, path)
	})

	t.Run("pre-existing world-readable file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")
		if err := os.WriteFile(path, []byte("PORT=3000\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Apply(parse(t, "PORT=8080\nTOKEN=changeme\n"), path); err != nil {
			t.Fatal(err)
		}
		assertMode(t, path)
	})
}

func assertMode(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != FileMode {
		t.Errorf("mode = %04o, want %04o", got, FileMode)
	}
}

// Apply must not leave the caller's .env truncated or half-written, and must
// not leave its staging file behind.
func TestApplyLeavesNoTempFilesAndPreservesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	const original = "PORT=3000\nSECRET=placeholder-value\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Apply(parse(t, "PORT=8080\nTOKEN=changeme\n"), path); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), original) {
		t.Errorf("existing content was not preserved verbatim: %q", body)
	}
	if !strings.Contains(string(body), "TOKEN=changeme") {
		t.Errorf("missing key was not appended: %q", body)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("staging file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("directory = %d entries, want 1", len(entries))
	}
}

// A file that cannot be renamed into place must leave the original untouched
// rather than truncating it.
func TestApplyFailureLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", ".env")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	const original = "PORT=3000\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	// Make the directory unwritable so staging the replacement fails.
	if err := os.Chmod(filepath.Dir(path), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(path), 0o700) })

	if _, err := Apply(parse(t, "PORT=8080\nTOKEN=changeme\n"), path); err == nil {
		t.Skip("directory permissions are not enforced here (running as root?)")
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != original {
		t.Errorf("failed write clobbered the .env: %q, want %q", body, original)
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
