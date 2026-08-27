package sshconfig

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// configureCommandForBackground prevents short-lived OpenSSH configuration
// probes (ssh -G) from creating a console window for every discovered Host.
func configureCommandForBackground(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
