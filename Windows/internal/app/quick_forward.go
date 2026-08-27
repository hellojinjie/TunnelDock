package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

// TemporaryTunnelConnector is implemented by tunnel.Manager. Keeping this
// narrow interface lets the Walk layer request a connection without owning the
// lifecycle state machine.
type TemporaryTunnelConnector interface {
	ConnectTemporary(context.Context, model.TunnelDefinition) (string, error)
}

type FocusTarget int

const (
	FocusNone FocusTarget = iota
	FocusLocalPort
)

type QuickForward struct {
	RemoteHost             string
	RemotePort             string
	LocalAddress           string
	LocalPort              string
	WebProtocol            model.TunnelProtocol
	AdvancedExpanded       bool
	Focus                  FocusTarget
	localPortFollowsRemote bool
}

func NewQuickForward() *QuickForward {
	quick := &QuickForward{}
	quick.Reset()
	return quick
}

func (q *QuickForward) SetRemotePort(value string) {
	q.RemotePort = value
	if q.localPortFollowsRemote {
		q.LocalPort = value
	}
}

func (q *QuickForward) SetLocalPort(value string) {
	q.LocalPort = value
	q.localPortFollowsRemote = false
}

func (q *QuickForward) LocalPortFollowsRemote() bool { return q.localPortFollowsRemote }

func (q *QuickForward) HandlePortConflict() {
	q.AdvancedExpanded = true
	q.Focus = FocusLocalPort
}

// TunnelDefinition validates the text fields owned by the Quick Forward form
// and creates the temporary definition passed to TunnelManager.
func (q *QuickForward) TunnelDefinition(hostAlias string) (model.TunnelDefinition, error) {
	remotePort, err := parsePort(q.RemotePort, "remote")
	if err != nil {
		return model.TunnelDefinition{}, err
	}
	localPort, err := parsePort(q.LocalPort, "local")
	if err != nil {
		return model.TunnelDefinition{}, err
	}

	definition := model.TunnelDefinition{
		HostAlias:    hostAlias,
		RemoteHost:   q.RemoteHost,
		RemotePort:   remotePort,
		LocalAddress: q.LocalAddress,
		LocalPort:    localPort,
		WebProtocol:  q.WebProtocol,
	}
	if err := definition.Validate(); err != nil {
		return model.TunnelDefinition{}, err
	}
	return definition, nil
}

// Connect validates the form before delegating creation of the runtime-only
// tunnel to TunnelManager.
func (q *QuickForward) Connect(ctx context.Context, connector TemporaryTunnelConnector, hostAlias string) (string, error) {
	definition, err := q.TunnelDefinition(hostAlias)
	if err != nil {
		return "", err
	}
	return connector.ConnectTemporary(ctx, definition)
}

func (q *QuickForward) Reset() {
	q.RemoteHost = "127.0.0.1"
	q.RemotePort = ""
	q.LocalAddress = "127.0.0.1"
	q.LocalPort = ""
	q.WebProtocol = model.TunnelProtocolHTTP
	q.AdvancedExpanded = false
	q.Focus = FocusNone
	q.localPortFollowsRemote = true
}

func parsePort(value, field string) (uint16, error) {
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("%s port must be between 1 and 65535", field)
	}
	return uint16(port), nil
}
