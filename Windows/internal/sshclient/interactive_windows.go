package sshclient

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unicode"

	"golang.org/x/sys/windows"
)

var findInteractiveOpenSSH = LocateOpenSSH
var launchInteractiveSSH = launchInteractiveSSHProcess

// CanStartInteractiveSSH permits aliases that can safely be passed as a
// standalone ssh.exe destination argument. TunnelDock never uses a shell to
// construct this command.
func CanStartInteractiveSSH(alias string) bool {
	if alias == "" || strings.HasPrefix(alias, "-") {
		return false
	}
	for _, character := range alias {
		if !(unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune("._-", character)) {
			return false
		}
	}
	return true
}

func InteractiveSSHCommand(alias string) string { return "ssh " + alias }

// StartInteractiveSSH opens a dedicated terminal and starts ssh.exe without
// BatchMode. It is intentionally used only after a connection failure that
// needs interactive authentication or host-key confirmation.
func StartInteractiveSSH(alias string) error {
	if !CanStartInteractiveSSH(alias) {
		return fmt.Errorf("cannot open an interactive SSH terminal for Host %q", alias)
	}
	executable, err := findInteractiveOpenSSH()
	if err != nil {
		return err
	}
	return launchInteractiveSSH(executable, alias)
}

func launchInteractiveSSHProcess(executable, alias string) error {
	command := exec.Command(executable, alias)
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE}
	return command.Start()
}
