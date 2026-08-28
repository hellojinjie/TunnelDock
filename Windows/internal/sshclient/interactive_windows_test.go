package sshclient

import (
	"errors"
	"testing"
)

func TestCanStartInteractiveSSHAllowsNormalHostAliases(t *testing.T) {
	for _, alias := range []string{"gpu", "build-server", "dev_01", "node.example"} {
		if !CanStartInteractiveSSH(alias) {
			t.Fatalf("CanStartInteractiveSSH(%q) = false", alias)
		}
	}
	for _, alias := range []string{"", "-oProxyCommand", "dev host", "dev;exit"} {
		if CanStartInteractiveSSH(alias) {
			t.Fatalf("CanStartInteractiveSSH(%q) = true", alias)
		}
	}
}

func TestStartInteractiveSSHUsesStandaloneSSHArgument(t *testing.T) {
	originalFind, originalLaunch := findInteractiveOpenSSH, launchInteractiveSSH
	t.Cleanup(func() {
		findInteractiveOpenSSH = originalFind
		launchInteractiveSSH = originalLaunch
	})
	findInteractiveOpenSSH = func() (string, error) { return `C:\Windows\System32\OpenSSH\ssh.exe`, nil }
	var executable, alias string
	launchInteractiveSSH = func(gotExecutable, gotAlias string) error {
		executable, alias = gotExecutable, gotAlias
		return nil
	}
	if err := StartInteractiveSSH("gpu"); err != nil {
		t.Fatalf("StartInteractiveSSH() error = %v", err)
	}
	if executable != `C:\Windows\System32\OpenSSH\ssh.exe` || alias != "gpu" {
		t.Fatalf("interactive command = %q %q", executable, alias)
	}

	findInteractiveOpenSSH = func() (string, error) { return "", errors.New("not installed") }
	if err := StartInteractiveSSH("gpu"); err == nil {
		t.Fatal("StartInteractiveSSH() succeeded without ssh.exe")
	}
}
