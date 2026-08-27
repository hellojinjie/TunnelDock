package sshclient

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocatePrefersWindowsSystemOpenSSH(t *testing.T) {
	windir := t.TempDir()
	systemSSH := filepath.Join(windir, "System32", "OpenSSH", "ssh.exe")
	writeEmptyFile(t, systemSSH)
	got, err := locateOpenSSH(windir, func(string) (string, error) { return `C:\tools\ssh.exe`, nil })
	if err != nil || got != systemSSH {
		t.Fatalf("locateOpenSSH() = %q, %v; want %q", got, err, systemSSH)
	}
}

func TestLocateFallsBackToPATHAndReturnsClearMissingError(t *testing.T) {
	got, err := locateOpenSSH(t.TempDir(), func(name string) (string, error) {
		if name != "ssh.exe" {
			t.Fatalf("LookPath(%q)", name)
		}
		return `C:\tools\ssh.exe`, nil
	})
	if err != nil || got != `C:\tools\ssh.exe` {
		t.Fatalf("PATH fallback = %q, %v", got, err)
	}

	_, err = locateOpenSSH(t.TempDir(), func(string) (string, error) { return "", errors.New("missing") })
	if !errors.Is(err, ErrOpenSSHNotInstalled) || err.Error() != OpenSSHNotInstalledMessage {
		t.Fatalf("missing error = %v", err)
	}
}

func writeEmptyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}
