package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestPresentHostRowsShowsIdentityAndAvailability(t *testing.T) {
	rows := PresentHostRows([]model.SSHHost{
		{Alias: "gpu", User: "alice", Hostname: "gpu.example", Port: 22, Availability: model.HostAvailable},
		{Alias: "broken", Hostname: "broken", Port: 22, Availability: model.HostConfigurationError},
	}, map[string]bool{"gpu": true})
	if len(rows) != 2 {
		t.Fatalf("len(PresentHostRows()) = %d", len(rows))
	}
	if rows[0].ID != "gpu" || rows[0].Title != "gpu" || !rows[0].Active || rows[0].Availability != model.HostAvailable {
		t.Fatalf("ready row = %#v", rows[0])
	}
	if rows[1].Availability != model.HostConfigurationError {
		t.Fatalf("error row = %#v", rows[1])
	}
}

func TestHostForAliasDoesNotDependOnTableRowOrder(t *testing.T) {
	hosts := []model.SSHHost{
		{Alias: "zebra", Hostname: "zebra.example"},
		{Alias: "alpha", Hostname: "alpha.example"},
	}
	host, found := HostForAlias(hosts, "alpha")
	if !found || host.Hostname != "alpha.example" {
		t.Fatalf("HostForAlias() = %#v, %v", host, found)
	}
}

func TestPresentMissingHostRowsOnlyShowsSavedAliasesAbsentFromConfig(t *testing.T) {
	rows := PresentMissingHostRows(
		[]model.SSHHost{{Alias: "gpu"}},
		[]model.TunnelRuntime{
			{Definition: model.TunnelDefinition{HostAlias: "gpu"}},
			{Definition: model.TunnelDefinition{HostAlias: "old"}},
			{Definition: model.TunnelDefinition{HostAlias: "gone"}},
			{Temporary: true, Definition: model.TunnelDefinition{HostAlias: "temporary"}},
		},
		"old",
	)
	if len(rows) != 1 || rows[0].ID != "old" || !rows[0].Missing {
		t.Fatalf("PresentMissingHostRows() = %#v", rows)
	}
}
