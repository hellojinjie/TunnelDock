package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestTunnelListRowsSeparatesSavedAndTemporaryTunnels(t *testing.T) {
	rows := TunnelListRows([]model.TunnelRuntime{
		{ID: "saved", Definition: model.TunnelDefinition{RemotePort: 8888, LocalPort: 8888}, State: model.StateConnected},
		{ID: "temporary", Temporary: true, Definition: model.TunnelDefinition{RemotePort: 3000, LocalPort: 13000}, State: model.StateDisconnected},
	})
	if len(rows.Saved) != 1 || rows.Saved[0] != "8888 — Connected" || len(rows.Temporary) != 1 || rows.Temporary[0] != "13000 → 3000 — Disconnected" {
		t.Fatalf("TunnelListRows() = %#v", rows)
	}
}
