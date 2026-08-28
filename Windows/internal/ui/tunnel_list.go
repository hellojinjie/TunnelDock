package ui

import (
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
)

type TunnelRows struct {
	Saved     []string
	Temporary []string
}

type TunnelTableRow struct {
	RuntimeID string
	Name      string
	Forward   string
	Status    string
}

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

func TunnelRowTexts(runtimes []model.TunnelRuntime) []string {
	rows := make([]string, 0, len(runtimes))
	for _, runtime := range runtimes {
		rows = append(rows, fmt.Sprintf("%s — %s", runtime.DisplayName(), tunnelStateText(runtime.State)))
	}
	return rows
}

func TunnelTableRows(runtimes []model.TunnelRuntime) []*TunnelTableRow {
	rows := make([]*TunnelTableRow, 0, len(runtimes))
	for _, runtime := range runtimes {
		rows = append(rows, &TunnelTableRow{
			RuntimeID: runtime.ID,
			Name:      runtime.DisplayName(),
			Forward:   fmt.Sprintf("%s:%d → %s:%d", runtime.Definition.LocalAddress, runtime.Definition.LocalPort, runtime.Definition.RemoteHost, runtime.Definition.RemotePort),
			Status:    tunnelStateText(runtime.State),
		})
	}
	return rows
}

// TunnelForRuntimeID resolves a sorted table row to its runtime snapshot.
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

func TunnelListRows(runtimes []model.TunnelRuntime) TunnelRows {
	rows := TunnelRows{}
	for _, runtime := range runtimes {
		row := fmt.Sprintf("%s — %s", runtime.DisplayName(), tunnelStateText(runtime.State))
		if runtime.Temporary {
			rows.Temporary = append(rows.Temporary, row)
		} else {
			rows.Saved = append(rows.Saved, row)
		}
	}
	return rows
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
