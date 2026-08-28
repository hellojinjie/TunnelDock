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

func TestTunnelsForHostKeepsOnlySelectedHostRuntimes(t *testing.T) {
	runtimes := []model.TunnelRuntime{
		{ID: "one", Definition: model.TunnelDefinition{HostAlias: "gpu"}},
		{ID: "two", Definition: model.TunnelDefinition{HostAlias: "nas"}},
		{ID: "three", Definition: model.TunnelDefinition{HostAlias: "gpu"}, Temporary: true},
	}
	selected := TunnelsForHost(runtimes, "gpu")
	if len(selected) != 2 || selected[0].ID != "one" || selected[1].ID != "three" {
		t.Fatalf("TunnelsForHost() = %#v", selected)
	}
}

func TestTunnelTableRowsShowForwardAndState(t *testing.T) {
	rows := TunnelTableRows([]model.TunnelRuntime{{
		ID: "one", Definition: model.TunnelDefinition{HostAlias: "gpu", Name: stringPtr("Jupyter"), LocalAddress: "127.0.0.1", LocalPort: 8888, RemoteHost: "127.0.0.1", RemotePort: 8888}, State: model.StateConnected,
	}})
	if len(rows) != 1 || rows[0].RuntimeID != "one" || rows[0].Host != "gpu" || rows[0].Name != "Jupyter" || rows[0].Forward != "127.0.0.1:8888 → 127.0.0.1:8888" || rows[0].Status != "Connected" {
		t.Fatalf("TunnelTableRows() = %#v", rows)
	}
}

func TestTunnelForRuntimeIDDoesNotDependOnTableRowOrder(t *testing.T) {
	runtimes := []model.TunnelRuntime{
		{ID: "zebra", Definition: model.TunnelDefinition{HostAlias: "gpu"}},
		{ID: "alpha", Definition: model.TunnelDefinition{HostAlias: "gpu"}},
	}
	runtime, found := TunnelForRuntimeID(runtimes, "alpha")
	if !found || runtime.ID != "alpha" {
		t.Fatalf("TunnelForRuntimeID() = %#v, %v", runtime, found)
	}
}

func stringPtr(value string) *string { return &value }
