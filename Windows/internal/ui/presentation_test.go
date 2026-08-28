package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestPresentTunnelRowsOwnsInlineActions(t *testing.T) {
	rows := PresentTunnelRows(
		[]model.TunnelRuntime{{
			ID: "jupyter",
			Definition: model.TunnelDefinition{
				HostAlias: "gpu", Name: stringPtr("Jupyter"),
				LocalAddress: "127.0.0.1", LocalPort: 8888,
				RemoteHost: "127.0.0.1", RemotePort: 8888,
			},
			State: model.StateConnected,
		}},
		[]model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}},
	)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ID != "jupyter" || row.Name != "Jupyter" || row.StateText != "Connected" {
		t.Fatalf("row = %#v", row)
	}
	if row.PrimaryAction != TunnelRowDisconnect || !row.ShowBrowser || !row.ShowMore {
		t.Fatalf("actions = %#v", row)
	}
}

func TestPresentFailedTunnelIncludesInlineError(t *testing.T) {
	rows := PresentTunnelRows([]model.TunnelRuntime{{
		ID: "failed", State: model.StateFailed, LastError: "Permission denied",
		Definition: model.TunnelDefinition{HostAlias: "gpu", LocalPort: 9000, RemotePort: 9000},
	}}, []model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}})
	if rows[0].ErrorText != "Permission denied" || rows[0].PrimaryAction != TunnelRowConnect {
		t.Fatalf("row = %#v", rows[0])
	}
}

func TestPresentHostRowsShowsActivityWithoutChangingIdentity(t *testing.T) {
	rows := PresentHostRows(
		[]model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}},
		map[string]bool{"gpu": true},
	)
	if len(rows) != 1 || rows[0].ID != "gpu" || !rows[0].Active || rows[0].Missing {
		t.Fatalf("rows = %#v", rows)
	}
}
