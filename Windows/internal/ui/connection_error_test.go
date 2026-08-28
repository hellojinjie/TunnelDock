package ui

import (
	"fmt"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
)

func TestPresentConnectionErrorIncludesReasonActionAndDetails(t *testing.T) {
	presentation := PresentConnectionError(sshclient.NewConnectionFailure("Permission denied (publickey).", sshclient.ErrProcessExited))
	if presentation.Title != "Connection failed" || presentation.Summary != "Authentication failed." || presentation.Action == "" || presentation.Details != "Permission denied (publickey)." || !presentation.RequiresInteractiveSSH {
		t.Fatalf("PresentConnectionError() = %#v", presentation)
	}
}

func TestPresentConnectionErrorExplainsLocalPortConflict(t *testing.T) {
	presentation := PresentConnectionError(fmt.Errorf("%w: occupied", tunnel.ErrPortUnavailable))
	if presentation.Title != "Local port unavailable" || presentation.Action == "" {
		t.Fatalf("PresentConnectionError() = %#v", presentation)
	}
}
