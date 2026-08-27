package sshclient

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

const OpenSSHNotInstalledMessage = "OpenSSH Client is not installed.\nInstall the Windows OpenSSH Client feature and restart TunnelDock."

var ErrOpenSSHNotInstalled = errors.New(OpenSSHNotInstalledMessage)

func LocateOpenSSH() (string, error) {
	return locateOpenSSH(os.Getenv("WINDIR"), exec.LookPath)
}

func locateOpenSSH(windowsDirectory string, lookPath func(string) (string, error)) (string, error) {
	if windowsDirectory != "" {
		candidate := filepath.Join(windowsDirectory, "System32", "OpenSSH", "ssh.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if candidate, err := lookPath("ssh.exe"); err == nil {
		return candidate, nil
	}
	return "", ErrOpenSSHNotInstalled
}
