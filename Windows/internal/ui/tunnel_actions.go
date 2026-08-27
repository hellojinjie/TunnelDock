package ui

import (
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

func TunnelBrowserURL(definition model.TunnelDefinition) string {
	protocol := definition.WebProtocol
	if protocol != model.TunnelProtocolHTTPS {
		protocol = model.TunnelProtocolHTTP
	}
	return fmt.Sprintf("%s://127.0.0.1:%d", protocol, definition.LocalPort)
}
