package app

import (
	"reflect"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TestFilterHostsMatchesAllEffectiveFieldsWithoutReordering(t *testing.T) {
	hosts := []model.SSHHost{
		{Alias: "first", Hostname: "10.0.0.1", User: "root", Port: 22, ConfigOrder: 0},
		{Alias: "GPU-Server", Hostname: "compute.internal", User: "alice", Port: 2222, ConfigOrder: 1},
		{Alias: "third", Hostname: "gpu.example", User: "ROOT", Port: 2200, ConfigOrder: 2},
	}
	tests := []struct {
		query string
		want  []string
	}{
		{query: "root", want: []string{"first", "third"}},
		{query: "GPU", want: []string{"GPU-Server", "third"}},
		{query: "2222", want: []string{"GPU-Server"}},
		{query: "", want: []string{"first", "GPU-Server", "third"}},
	}
	for _, tt := range tests {
		got := FilterHosts(hosts, tt.query)
		aliases := make([]string, len(got))
		for index := range got {
			aliases[index] = got[index].Alias
		}
		if !reflect.DeepEqual(aliases, tt.want) {
			t.Errorf("FilterHosts(%q) = %#v, want %#v", tt.query, aliases, tt.want)
		}
	}
}

func TestAppModelFiltersHostsUsingCurrentQuery(t *testing.T) {
	application := NewModel()
	application.SetHosts([]model.SSHHost{{Alias: "gpu"}, {Alias: "nas"}})
	application.SetSearchQuery("AS")
	if got := application.FilteredHosts(); len(got) != 1 || got[0].Alias != "nas" {
		t.Fatalf("FilteredHosts() = %#v", got)
	}
}
