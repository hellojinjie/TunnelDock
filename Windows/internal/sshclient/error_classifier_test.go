package sshclient

import (
	"errors"
	"testing"
)

func TestClassifyOpenSSHError(t *testing.T) {
	tests := []struct {
		name        string
		stderr      string
		launchErr   error
		wantKind    ErrorKind
		wantMessage string
	}{
		{name: "local port", stderr: "bind [127.0.0.1]:8888: Address already in use", wantKind: ErrorLocalPortInUse, wantMessage: "Local port is already in use."},
		{name: "authentication", stderr: "Permission denied (publickey,password,keyboard-interactive).", wantKind: ErrorAuthenticationFailed, wantMessage: "Authentication failed."},
		{name: "host verification", stderr: "No ED25519 host key is known for server and you have requested strict checking.\nHost key verification failed.", wantKind: ErrorHostVerificationRequired, wantMessage: "Host verification required."},
		{name: "host missing", stderr: "Host alias is no longer present in the SSH configuration.", wantKind: ErrorHostNotFound, wantMessage: "Host not found."},
		{name: "configuration", stderr: "C:\\Users\\me\\.ssh\\config line 4: Bad configuration option: controlfoo", wantKind: ErrorSSHConfiguration, wantMessage: "SSH configuration error."},
		{name: "timeout", stderr: "ssh: connect to host server port 22: Connection timed out", wantKind: ErrorConnectionTimedOut, wantMessage: "Connection timed out."},
		{name: "resolve", stderr: "ssh: Could not resolve hostname server: No such host is known.", wantKind: ErrorCouldNotResolveHost, wantMessage: "Could not resolve host."},
		{name: "refused", stderr: "ssh: connect to host server port 22: Connection refused", wantKind: ErrorSSHServerRefused, wantMessage: "Connection refused by SSH server."},
		{name: "not installed", launchErr: ErrOpenSSHNotInstalled, wantKind: ErrorOpenSSHNotInstalled, wantMessage: OpenSSHNotInstalledMessage},
		{name: "unknown", stderr: "ssh exited with status 255", launchErr: errors.New("exit status 255"), wantKind: ErrorUnexpectedExit, wantMessage: "SSH connection failed.\nSee Log for details."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyOpenSSHError(tt.stderr, tt.launchErr)
			if got.Kind != tt.wantKind || got.Message != tt.wantMessage {
				t.Fatalf("ClassifyOpenSSHError() = %#v, want kind %v message %q", got, tt.wantKind, tt.wantMessage)
			}
			if got.RawStderr != tt.stderr {
				t.Fatalf("RawStderr = %q, want %q", got.RawStderr, tt.stderr)
			}
		})
	}
}
