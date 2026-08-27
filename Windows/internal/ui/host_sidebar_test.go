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
