package ui

import (
	"strings"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

type HostTableRow struct {
	Alias      string
	Connection string
	Status     string
}

func HostTableRows(hosts []model.SSHHost) []*HostTableRow {
	rows := make([]*HostTableRow, 0, len(hosts))
	for _, host := range hosts {
		detail := HostDetailFor(&host)
		rows = append(rows, &HostTableRow{Alias: host.Alias, Connection: detail.Connection, Status: hostAvailabilityText(host.Availability)})
	}
	return rows
}

// MissingHostTableRows mirrors the macOS sidebar: only saved tunnel aliases
// that are absent from the current SSH config are shown, and search applies to
// the alias without changing the order of the saved definitions.
func MissingHostTableRows(hosts []model.SSHHost, runtimes []model.TunnelRuntime, query string) []*HostTableRow {
	known := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		known[host.Alias] = struct{}{}
	}
	seen := make(map[string]struct{})
	query = strings.ToLower(query)
	rows := make([]*HostTableRow, 0)
	for _, runtime := range runtimes {
		if runtime.Temporary || runtime.Definition.HostAlias == "" {
			continue
		}
		alias := runtime.Definition.HostAlias
		if _, exists := known[alias]; exists {
			continue
		}
		if _, exists := seen[alias]; exists || (query != "" && !strings.Contains(strings.ToLower(alias), query)) {
			continue
		}
		seen[alias] = struct{}{}
		rows = append(rows, &HostTableRow{Alias: alias, Connection: "SSH host not found", Status: "Unavailable"})
	}
	return rows
}

func hostAvailabilityText(availability model.HostAvailability) string {
	switch availability {
	case model.HostAvailable:
		return "Ready"
	case model.HostConfigurationError:
		return "Configuration error"
	case model.HostMissing:
		return "Unavailable"
	default:
		return "Unavailable"
	}
}

// PartitionHosts separates unavailable historical entries from current SSH
// configuration entries while preserving their resolved configuration order.
func PartitionHosts(hosts []model.SSHHost) (available, missing []model.SSHHost) {
	for _, host := range hosts {
		if host.Availability == model.HostMissing {
			missing = append(missing, host)
			continue
		}
		available = append(available, host)
	}
	return available, missing
}

func hostAliases(hosts []model.SSHHost) []string {
	aliases := make([]string, len(hosts))
	for i, host := range hosts {
		aliases[i] = host.Alias
	}
	return aliases
}
