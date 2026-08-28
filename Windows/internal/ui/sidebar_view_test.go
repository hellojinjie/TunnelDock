package ui

import "testing"

func TestSidebarStatePreservesSelectionWhenAliasRemains(t *testing.T) {
	state := newSidebarState()
	state.SetSelected("gpu")
	state.Apply([]HostRowPresentation{{ID: "nas"}, {ID: "gpu"}})
	if state.Selected() != "gpu" {
		t.Fatalf("selected = %q", state.Selected())
	}
	state.Apply([]HostRowPresentation{{ID: "nas"}})
	if state.Selected() != allTunnelsPaneID {
		t.Fatalf("selected after removal = %q", state.Selected())
	}
}

func TestSidebarStateKeepsAllTunnelsSelectedAcrossRefresh(t *testing.T) {
	state := newSidebarState()
	state.Apply([]HostRowPresentation{{ID: "gpu"}})
	if state.Selected() != allTunnelsPaneID {
		t.Fatalf("selected = %q", state.Selected())
	}
}
