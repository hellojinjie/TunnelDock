package ui

import "github.com/hellojinjie/TunnelDock/Windows/internal/model"

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
