package sshclient

import (
	"reflect"
	"testing"
)

func TestBuildTunnelCommandReturnsExactArgumentArray(t *testing.T) {
	got := BuildTunnelCommand(`C:\Windows\System32\OpenSSH\ssh.exe`, `C:\runtime\ssh_config`, "127.0.0.1:8888:127.0.0.1:8888", "gpu")
	want := Command{
		Executable: `C:\Windows\System32\OpenSSH\ssh.exe`,
		Args: []string{
			"-F", `C:\runtime\ssh_config`,
			"-N", "-T", "-n",
			"-L", "127.0.0.1:8888:127.0.0.1:8888",
			"-o", "ExitOnForwardFailure=yes",
			"-o", "ServerAliveInterval=15",
			"-o", "ServerAliveCountMax=3",
			"-o", "BatchMode=yes",
			"-o", "StrictHostKeyChecking=yes",
			"gpu",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTunnelCommand() = %#v, want %#v", got, want)
	}
}
