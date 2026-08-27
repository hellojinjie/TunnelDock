package ui

import (
	"fmt"
	"strconv"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
)

func promptTunnelEdit(owner walk.Form, initial model.TunnelDefinition) (model.TunnelDefinition, bool, error) {
	dialog, err := walk.NewDialogWithFixedSize(owner)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	defer dialog.Dispose()
	if err := dialog.SetTitle("Edit Tunnel"); err != nil {
		return model.TunnelDefinition{}, false, err
	}
	if err := dialog.SetSize(walk.Size{Width: 420, Height: 330}); err != nil {
		return model.TunnelDefinition{}, false, err
	}
	if err := dialog.SetLayout(walk.NewVBoxLayout()); err != nil {
		return model.TunnelDefinition{}, false, err
	}
	fields := make(map[string]*walk.LineEdit)
	for _, field := range []struct {
		label string
		key   string
		value string
	}{
		{"Host alias", "host", initial.HostAlias},
		{"Remote host", "remoteHost", initial.RemoteHost},
		{"Remote port", "remotePort", strconv.Itoa(int(initial.RemotePort))},
		{"Local address", "localAddress", initial.LocalAddress},
		{"Local port", "localPort", strconv.Itoa(int(initial.LocalPort))},
	} {
		input, err := addDialogField(dialog, field.label, field.value)
		if err != nil {
			return model.TunnelDefinition{}, false, err
		}
		fields[field.key] = input
	}
	protocolRow, err := walk.NewComposite(dialog)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	_ = protocolRow.SetLayout(walk.NewHBoxLayout())
	protocolLabel, err := walk.NewLabel(protocolRow)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	_ = protocolLabel.SetText("Browser protocol")
	protocol, err := walk.NewComboBox(protocolRow)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	if err := protocol.SetModel([]string{"http", "https"}); err != nil {
		return model.TunnelDefinition{}, false, err
	}
	if initial.WebProtocol == model.TunnelProtocolHTTPS {
		_ = protocol.SetCurrentIndex(1)
	} else {
		_ = protocol.SetCurrentIndex(0)
	}
	buttons, err := walk.NewComposite(dialog)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	buttons.SetLayout(walk.NewHBoxLayout())
	save, err := walk.NewPushButton(buttons)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	_ = save.SetText("Save")
	save.Clicked().Attach(dialog.Accept)
	cancel, err := walk.NewPushButton(buttons)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	_ = cancel.SetText("Cancel")
	cancel.Clicked().Attach(dialog.Cancel)
	if dialog.Run() != walk.DlgCmdOK {
		return model.TunnelDefinition{}, false, nil
	}
	remotePort, err := parseDialogPort("remote port", fields["remotePort"].Text())
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	localPort, err := parseDialogPort("local port", fields["localPort"].Text())
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	protocolValue := model.TunnelProtocolHTTP
	if protocol.CurrentIndex() == 1 {
		protocolValue = model.TunnelProtocolHTTPS
	}
	updated := initial
	updated.HostAlias = fields["host"].Text()
	updated.RemoteHost = fields["remoteHost"].Text()
	updated.RemotePort = remotePort
	updated.LocalAddress = fields["localAddress"].Text()
	updated.LocalPort = localPort
	updated.WebProtocol = protocolValue
	return updated, true, nil
}

func addDialogField(parent walk.Container, labelText, value string) (*walk.LineEdit, error) {
	row, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	if err := row.SetLayout(walk.NewHBoxLayout()); err != nil {
		return nil, err
	}
	label, err := walk.NewLabel(row)
	if err != nil {
		return nil, err
	}
	_ = label.SetText(labelText)
	input, err := walk.NewLineEdit(row)
	if err != nil {
		return nil, err
	}
	if err := input.SetText(value); err != nil {
		return nil, err
	}
	return input, nil
}

func parseDialogPort(name, value string) (uint16, error) {
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return uint16(port), nil
}
