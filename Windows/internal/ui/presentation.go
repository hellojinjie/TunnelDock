package ui

import (
	"fmt"
	"strings"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

type TunnelRowAction uint8

const (
	TunnelRowNoAction TunnelRowAction = iota
	TunnelRowConnect
	TunnelRowDisconnect
	TunnelRowOpenBrowser
	TunnelRowMore
)

type HostRowPresentation struct {
	ID           string
	Title        string
	Availability model.HostAvailability
	Active       bool
	Missing      bool
}

type TunnelRowPresentation struct {
	ID            string
	HostAlias     string
	Name          string
	Forward       string
	ErrorText     string
	State         model.TunnelState
	StateText     string
	Temporary     bool
	PrimaryAction TunnelRowAction
	PrimaryText   string
	ShowBrowser   bool
	ShowMore      bool
	CanConnect    bool
}

func PresentHostRows(hosts []model.SSHHost, activeAliases map[string]bool) []HostRowPresentation {
	rows := make([]HostRowPresentation, 0, len(hosts))
	for _, host := range hosts {
		rows = append(rows, HostRowPresentation{
			ID:           host.Alias,
			Title:        host.Alias,
			Availability: host.Availability,
			Active:       activeAliases[host.Alias],
			Missing:      host.Availability == model.HostMissing,
		})
	}
	return rows
}

func PresentMissingHostRows(hosts []model.SSHHost, runtimes []model.TunnelRuntime, query string) []HostRowPresentation {
	known := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		known[host.Alias] = true
	}
	query = strings.ToLower(strings.TrimSpace(query))
	seen := make(map[string]bool)
	rows := make([]HostRowPresentation, 0)
	for _, runtime := range runtimes {
		alias := runtime.Definition.HostAlias
		if runtime.Temporary || alias == "" || known[alias] || seen[alias] {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(alias), query) {
			continue
		}
		seen[alias] = true
		rows = append(rows, HostRowPresentation{
			ID:           alias,
			Title:        alias,
			Availability: model.HostMissing,
			Missing:      true,
		})
	}
	return rows
}

func PresentTunnelRows(runtimes []model.TunnelRuntime, hosts []model.SSHHost) []TunnelRowPresentation {
	available := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		available[host.Alias] = host.Availability == model.HostAvailable
	}
	rows := make([]TunnelRowPresentation, 0, len(runtimes))
	for _, runtime := range runtimes {
		canConnect := available[runtime.Definition.HostAlias]
		row := TunnelRowPresentation{
			ID:          runtime.ID,
			HostAlias:   runtime.Definition.HostAlias,
			Name:        runtime.DisplayName(),
			Forward:     fmt.Sprintf("%s:%d → %s:%d", runtime.Definition.LocalAddress, runtime.Definition.LocalPort, runtime.Definition.RemoteHost, runtime.Definition.RemotePort),
			ErrorText:   runtime.LastError,
			State:       runtime.State,
			StateText:   tunnelStateText(runtime.State),
			Temporary:   runtime.Temporary,
			ShowBrowser: runtime.State == model.StateConnected,
			ShowMore:    true,
			CanConnect:  canConnect,
		}
		switch runtime.State {
		case model.StateConnecting, model.StateConnected, model.StateReconnecting:
			row.PrimaryAction = TunnelRowDisconnect
			row.PrimaryText = "Disconnect"
		case model.StateDisconnected, model.StateFailed:
			if !runtime.Temporary && canConnect {
				row.PrimaryAction = TunnelRowConnect
				row.PrimaryText = "Connect"
			}
		}
		rows = append(rows, row)
	}
	return rows
}
