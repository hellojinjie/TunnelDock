package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestHostTableRowsShowConnectionAndAvailability(t *testing.T) {
	rows := HostTableRows([]model.SSHHost{
		{Alias: "gpu", User: "alice", Hostname: "gpu.example", Port: 22, Availability: model.HostAvailable},
		{Alias: "broken", Hostname: "broken", Port: 22, Availability: model.HostConfigurationError},
	})
	if len(rows) != 2 {
		t.Fatalf("len(HostTableRows()) = %d", len(rows))
	}
	if rows[0].Alias != "gpu" || rows[0].Connection != "alice@gpu.example:22" || rows[0].Status != "Ready" {
		t.Fatalf("ready row = %#v", rows[0])
	}
	if rows[1].Status != "Configuration error" {
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

func TestMissingHostTableRowsOnlyShowsSavedAliasesAbsentFromConfig(t *testing.T) {
	rows := MissingHostTableRows(
		[]model.SSHHost{{Alias: "gpu"}},
		[]model.TunnelRuntime{
			{Definition: model.TunnelDefinition{HostAlias: "gpu"}},
			{Definition: model.TunnelDefinition{HostAlias: "old"}},
			{Definition: model.TunnelDefinition{HostAlias: "gone"}},
			{Temporary: true, Definition: model.TunnelDefinition{HostAlias: "temporary"}},
		},
		"old",
	)
	if len(rows) != 1 || rows[0].Alias != "old" || rows[0].Status != "Unavailable" {
		t.Fatalf("MissingHostTableRows() = %#v", rows)
	}
}
