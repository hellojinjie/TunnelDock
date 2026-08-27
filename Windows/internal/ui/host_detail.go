package ui

import (
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

// HostDetail is the presentation-ready host identity displayed in the detail pane.
type HostDetail struct {
	Title      string
	Connection string
}

// HostDetailFor formats a host's effective identity without exposing UI controls
// to the application model.
func HostDetailFor(host *model.SSHHost) HostDetail {
	if host == nil {
		return HostDetail{
			Title:      "Select an SSH Host",
			Connection: "Choose a host from the sidebar.",
		}
	}

	connection := host.Hostname
	if host.User != "" {
		connection = host.User + "@" + connection
	}
	if host.Port != 0 {
		connection = fmt.Sprintf("%s:%d", connection, host.Port)
	}

	return HostDetail{Title: host.Alias, Connection: connection}
}
