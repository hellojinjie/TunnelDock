package ui

import (
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
)

// TunnelsForHost returns the manager's stable order restricted to the current
// host pane, matching the focused, host-first interaction used by the app.
func TunnelsForHost(runtimes []model.TunnelRuntime, alias string) []model.TunnelRuntime {
	if alias == "" {
		return nil
	}
	filtered := make([]model.TunnelRuntime, 0, len(runtimes))
	for _, runtime := range runtimes {
		if runtime.Definition.HostAlias == alias {
			filtered = append(filtered, runtime)
		}
	}
	return filtered
}

// TunnelForRuntimeID resolves a stable custom-row identity to its snapshot.
func TunnelForRuntimeID(runtimes []model.TunnelRuntime, runtimeID string) (model.TunnelRuntime, bool) {
	for _, runtime := range runtimes {
		if runtime.ID == runtimeID {
			return runtime, true
		}
	}
	return model.TunnelRuntime{}, false
}

func savedSnapshots(manager *tunnel.Manager) []model.TunnelRuntime {
	if manager == nil {
		return nil
	}
	var snapshots []model.TunnelRuntime
	for _, snapshot := range manager.Snapshots() {
		if !snapshot.Temporary {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots
}

func temporarySnapshots(manager *tunnel.Manager) []model.TunnelRuntime {
	if manager == nil {
		return nil
	}
	var snapshots []model.TunnelRuntime
	for _, snapshot := range manager.Snapshots() {
		if snapshot.Temporary {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots
}

func tunnelStateText(state model.TunnelState) string {
	switch state {
	case model.StateConnecting:
		return "Connecting"
	case model.StateConnected:
		return "Connected"
	case model.StateReconnecting:
		return "Reconnecting"
	case model.StateFailed:
		return "Failed"
	default:
		return "Disconnected"
	}
}
