package sshconfig

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestConfigureCommandForBackgroundSuppressesConsoleWindow(t *testing.T) {
	command := exec.Command("ssh.exe", "-G", "example")
	configureCommandForBackground(command)
	if command.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !command.SysProcAttr.HideWindow {
		t.Fatal("HideWindow = false")
	}
	if command.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NO_WINDOW", command.SysProcAttr.CreationFlags)
	}
}
