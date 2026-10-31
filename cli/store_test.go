package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(f, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run("store", "put", f, "-raw", dir, "-source", "test", "-url", "https://example.org/a")
	if code != ExitOK {
		t.Fatalf("put: %d %s", code, errs)
	}
	hash := strings.TrimSpace(out)
	if code, out, _ = run("store", "ls", "-raw", dir); code != ExitOK || out != "test\t1\n" {
		t.Fatalf("ls: %d %q", code, out)
	}
	if code, out, _ = run("store", "ls", "test", "-raw", dir); code != ExitOK || !strings.Contains(out, hash+"\t6\tlocal") {
		t.Fatalf("ls test: %d %q", code, out)
	}
	if code, out, _ = run("store", "cat", hash, "-raw", dir); code != ExitOK || out != "hello\n" {
		t.Fatalf("cat: %d %q", code, out)
	}
	if code, _, _ = run("store", "put", f, "-raw", dir); code != ExitUsage {
		t.Errorf("put with no source: want %d, got %d", ExitUsage, code)
	}
	if code, _, _ = run("store", "nothing", "-raw", dir); code != ExitUsage {
		t.Errorf("unknown: want %d, got %d", ExitUsage, code)
	}
}
