package app

import (
	"strconv"
	"strings"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func FilterHosts(hosts []model.SSHHost, query string) []model.SSHHost {
	query = strings.ToLower(query)
	if query == "" {
		return append([]model.SSHHost(nil), hosts...)
	}
	filtered := make([]model.SSHHost, 0, len(hosts))
	for _, host := range hosts {
		if strings.Contains(strings.ToLower(host.Alias), query) ||
			strings.Contains(strings.ToLower(host.Hostname), query) ||
			strings.Contains(strings.ToLower(host.User), query) ||
			strings.Contains(strconv.FormatUint(uint64(host.Port), 10), query) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}
