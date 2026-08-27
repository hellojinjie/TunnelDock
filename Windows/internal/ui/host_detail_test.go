package ui

import (
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestHostDetailForHost(t *testing.T) {
	detail := HostDetailFor(&model.SSHHost{
		Alias:    "gpu-build",
		User:     "ubuntu",
		Hostname: "192.0.2.20",
		Port:     2222,
	})

	if detail.Title != "gpu-build" {
		t.Fatalf("Title = %q, want gpu-build", detail.Title)
	}
	if detail.Connection != "ubuntu@192.0.2.20:2222" {
		t.Fatalf("Connection = %q, want ubuntu@192.0.2.20:2222", detail.Connection)
	}
}

func TestHostDetailForNoSelection(t *testing.T) {
	detail := HostDetailFor(nil)

	if detail.Title != "Select an SSH Host" {
		t.Fatalf("Title = %q, want selection prompt", detail.Title)
	}
	if detail.Connection != "Choose a host from the sidebar." {
		t.Fatalf("Connection = %q, want selection guidance", detail.Connection)
	}
}

func TestPartitionHosts(t *testing.T) {
	hosts := []model.SSHHost{
		{Alias: "prod", Availability: model.HostAvailable},
		{Alias: "old-laptop", Availability: model.HostMissing},
		{Alias: "broken", Availability: model.HostConfigurationError},
	}

	available, missing := PartitionHosts(hosts)
	if len(available) != 2 || available[0].Alias != "prod" || available[1].Alias != "broken" {
		t.Fatalf("available = %#v, want prod and broken", available)
	}
	if len(missing) != 1 || missing[0].Alias != "old-laptop" {
		t.Fatalf("missing = %#v, want old-laptop", missing)
	}
}
