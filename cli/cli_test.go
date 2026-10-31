package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, Env{Version: "test", Stdout: &out, Stderr: &errb})
	return code, out.String(), errb.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := run("version")
	if code != ExitOK || out != "om test\n" {
		t.Fatalf("got %d %q", code, out)
	}
}

func TestHelpListsEachCommand(t *testing.T) {
	code, out, _ := run("help")
	if code != ExitOK {
		t.Fatalf("exit code %d", code)
	}
	for name := range commands {
		if !strings.Contains(out, "  "+name) {
			t.Errorf("help does not list %s", name)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errs := run("frobnicate")
	if code != ExitUsage || !strings.Contains(errs, "frobnicate") {
		t.Fatalf("got %d %q", code, errs)
	}
}

func TestNoArguments(t *testing.T) {
	if code, _, _ := run(); code != ExitUsage {
		t.Fatalf("exit code %d", code)
	}
}
