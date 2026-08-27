package sshconfig_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	. "github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
)

func TestSanitizeRemovesForwardingAndControlDirectivesOnly(t *testing.T) {
	lines := []ExpandedLine{
		{Text: "# user comment"},
		{Text: "Host gpu"},
		{Text: "    HostName 192.0.2.10"},
		{Text: "    LocalForward 8888 localhost:8888"},
		{Text: "remoteFORWARD=9000 localhost:9000"},
		{Text: "    DynamicForward 1080"},
		{Text: "    ClearAllForwardings yes"},
		{Text: "    ControlMaster auto"},
		{Text: "    ControlPath ~/.ssh/control-%C"},
		{Text: "    ControlPersist 5m"},
		{Text: "    Include config.d/*"},
		{Text: "    IdentityFile ~/.ssh/id_ed25519"},
		{Text: "Match host gpu"},
		{Text: "    ProxyJump bastion"},
	}

	got := Sanitize(lines)
	want := strings.Join([]string{
		"# user comment",
		"Host gpu",
		"    HostName 192.0.2.10",
		"    IdentityFile ~/.ssh/id_ed25519",
		"Match host gpu",
		"    ProxyJump bastion",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("Sanitize() =\n%q\nwant\n%q", got, want)
	}
}

func TestSanitizedConfigPreservesEffectiveOpenSSHValues(t *testing.T) {
	ssh, err := sshclient.LocateOpenSSH()
	if err != nil {
		t.Skip(err)
	}
	directory := t.TempDir()
	identity := filepath.ToSlash(filepath.Join(directory, "id_ed25519"))
	lines := []ExpandedLine{
		{Text: "Host gpu"},
		{Text: "  HostName 192.0.2.10"},
		{Text: "  User alice"},
		{Text: "  Port 2222"},
		{Text: "  ProxyJump bastion"},
		{Text: "  IdentityFile " + identity},
		{Text: "  LocalForward 18888 127.0.0.1:8888"},
		{Text: "  ControlMaster auto"},
	}
	original := filepath.Join(directory, "original_config")
	sanitized := filepath.Join(directory, "sanitized_config")
	if err := os.WriteFile(original, []byte(strings.Join(lineTexts(lines), "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sanitized, []byte(Sanitize(lines)), 0o600); err != nil {
		t.Fatal(err)
	}

	originalValues := effectiveValues(t, ssh, original)
	sanitizedValues := effectiveValues(t, ssh, sanitized)
	for _, key := range []string{"hostname", "user", "port", "proxyjump", "identityfile"} {
		if !reflect.DeepEqual(sanitizedValues[key], originalValues[key]) {
			t.Errorf("effective %s changed: original %#v, sanitized %#v", key, originalValues[key], sanitizedValues[key])
		}
	}
}

func TestRuntimeConfigStoreCreatesAndCleansIsolatedConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")
	store := NewRuntimeConfigStore(root)
	path, err := store.Create("runtime-1", []ExpandedLine{{Text: "Host gpu"}, {Text: "  User alice"}})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if path != filepath.Join(root, "runtime-1", "ssh_config") {
		t.Fatalf("Create() path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "Host gpu\n  User alice\n" {
		t.Fatalf("runtime config = %q, %v", data, err)
	}
	if err := store.Remove("runtime-1"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("runtime directory still exists: %v", err)
	}
}

func TestRuntimeConfigStoreRejectsUnsafeRuntimeIDAndRemovesStaleDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")
	store := NewRuntimeConfigStore(root)
	if _, err := store.Create(`..\escape`, nil); err == nil {
		t.Fatal("Create() accepted unsafe runtime ID")
	}
	if err := os.MkdirAll(filepath.Join(root, "stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveStale(); err != nil {
		t.Fatalf("RemoveStale() error: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("runtime root entries = %#v, %v", entries, err)
	}
}

func lineTexts(lines []ExpandedLine) []string {
	values := make([]string, len(lines))
	for index, line := range lines {
		values[index] = line.Text
	}
	return values
}

func effectiveValues(t *testing.T, ssh, config string) map[string][]string {
	t.Helper()
	output, err := exec.Command(ssh, "-F", config, "-G", "gpu").Output()
	if err != nil {
		t.Fatalf("ssh -F %q -G gpu: %v", config, err)
	}
	values := make(map[string][]string)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.ToLower(fields[0])
			values[key] = append(values[key], strings.Join(fields[1:], " "))
		}
	}
	return values
}
