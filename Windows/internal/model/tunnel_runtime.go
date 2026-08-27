package model

type TunnelState int

const (
	StateDisconnected TunnelState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateFailed
)

type TunnelRuntime struct {
	ID                string
	Definition        TunnelDefinition
	Temporary         bool
	State             TunnelState
	DesiredConnection bool
	LastError         string
	LogLines          []string
}

func (r TunnelRuntime) DisplayName() string {
	return r.Definition.DisplayName()
}
