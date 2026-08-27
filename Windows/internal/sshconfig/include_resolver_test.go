package sshconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIncludeResolverExpandsNestedGlobInTextOrder(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	writeTestFile(t, filepath.Join(sshDir, "config.d", "b"), "Host beta\n")
	writeTestFile(t, filepath.Join(sshDir, "config.d", "a"), "Host alpha\nInclude common\n")
	writeTestFile(t, filepath.Join(sshDir, "common"), "Host common\n")
	root := filepath.Join(sshDir, "config")
	writeTestFile(t, root, "Host before\nInclude config.d/*\nHost after\n")

	expanded, err := NewIncludeResolver(sshDir).Resolve(root)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	want := []string{"before", "alpha", "common", "beta", "after"}
	if got := (Scanner{}).DiscoverAliases(expanded.Lines); !reflect.DeepEqual(got, want) {
		t.Fatalf("aliases = %#v, want %#v", got, want)
	}
	for _, line := range expanded.Lines {
		if line.SourcePath == "" || line.LineNumber < 1 {
			t.Fatalf("line lacks source metadata: %#v", line)
		}
	}
}

func TestIncludeResolverSupportsTildeAbsoluteAndCyclePrevention(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	abs := filepath.Join(home, "absolute.conf")
	writeTestFile(t, abs, "Host absolute\n")
	writeTestFile(t, filepath.Join(home, "tilde.conf"), "Host tilde\n")
	writeTestFile(t, filepath.Join(sshDir, "a"), "Host a\nInclude b\n")
	writeTestFile(t, filepath.Join(sshDir, "b"), "Host b\nInclude a\n")
	root := filepath.Join(sshDir, "config")
	writeTestFile(t, root, "Include a\nInclude ~/tilde.conf\nInclude \""+filepath.ToSlash(abs)+"\"\n")

	expanded, err := NewIncludeResolver(sshDir).Resolve(root)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	want := []string{"a", "b", "tilde", "absolute"}
	if got := (Scanner{}).DiscoverAliases(expanded.Lines); !reflect.DeepEqual(got, want) {
		t.Fatalf("aliases = %#v, want %#v", got, want)
	}
	if len(expanded.Diagnostics) != 1 || expanded.Diagnostics[0].Kind != DiagnosticIncludeCycle {
		t.Fatalf("diagnostics = %#v, want one include cycle", expanded.Diagnostics)
	}
}

func TestIncludeResolverMissingConfigReturnsEmptyResult(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	expanded, err := NewIncludeResolver(sshDir).Resolve(filepath.Join(sshDir, "config"))
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if len(expanded.Lines) != 0 {
		t.Fatalf("Lines = %#v, want empty", expanded.Lines)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
