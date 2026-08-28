package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

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

func TestPresentTunnelRowsShowsForwardAndState(t *testing.T) {
	rows := PresentTunnelRows([]model.TunnelRuntime{{
		ID: "one", Definition: model.TunnelDefinition{HostAlias: "gpu", Name: stringPtr("Jupyter"), LocalAddress: "127.0.0.1", LocalPort: 8888, RemoteHost: "127.0.0.1", RemotePort: 8888}, State: model.StateConnected,
	}}, []model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}})
	if len(rows) != 1 || rows[0].ID != "one" || rows[0].HostAlias != "gpu" || rows[0].Name != "Jupyter" || rows[0].Forward != "127.0.0.1:8888 → 127.0.0.1:8888" || rows[0].StateText != "Connected" {
		t.Fatalf("PresentTunnelRows() = %#v", rows)
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
