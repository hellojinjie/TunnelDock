package app

import "github.com/hellojinjie/TunnelDock/Windows/internal/model"

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
