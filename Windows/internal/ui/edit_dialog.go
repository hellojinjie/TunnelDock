package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
)

func promptTunnelEdit(owner walk.Form, env *UIEnvironment, initial model.TunnelDefinition) (model.TunnelDefinition, bool, error) {
	shell, err := NewDialogShell(owner, env, DialogSpec{
		Title: "Edit Tunnel", Description: "Update the forwarding endpoints for this saved tunnel.",
		PrimaryText: "Save", Size: walk.Size{Width: 480, Height: 470},
	})
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	defer shell.Dispose()
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 14, VNear: 12, HFar: 14, VFar: 12})
	layout.SetSpacing(8)
	if err := shell.Content.SetLayout(layout); err != nil {
		return model.TunnelDefinition{}, false, err
	}
	fields := make(map[string]*walk.LineEdit)
	for _, field := range []struct{ label, key, value string }{
		{"Host alias", "host", initial.HostAlias},
		{"Remote host", "remoteHost", initial.RemoteHost},
		{"Remote port", "remotePort", strconv.Itoa(int(initial.RemotePort))},
		{"Local address", "localAddress", initial.LocalAddress},
		{"Local port", "localPort", strconv.Itoa(int(initial.LocalPort))},
	} {
		input, fieldErr := addDialogField(shell.Content, field.label, field.value)
		if fieldErr != nil {
			return model.TunnelDefinition{}, false, fieldErr
		}
		fields[field.key] = input
	}
	protocolRow, err := walk.NewComposite(shell.Content)
	if err != nil {
		return model.TunnelDefinition{}, false, err
	}
	protocolLayout := walk.NewHBoxLayout()
	protocolLayout.SetMargins(walk.Margins{})
	protocolLayout.SetSpacing(8)
	_ = protocolRow.SetLayout(protocolLayout)
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
	updated := initial
	accepted := false
	shell.Primary.Clicked().Attach(func() {
		remotePort, parseErr := parseDialogPort("Remote port", fields["remotePort"].Text())
		if parseErr != nil {
			shell.SetValidation(parseErr.Error(), fields["remotePort"])
			return
		}
		localPort, parseErr := parseDialogPort("Local port", fields["localPort"].Text())
		if parseErr != nil {
			shell.SetValidation(parseErr.Error(), fields["localPort"])
			return
		}
		updated = initial
		updated.HostAlias = strings.TrimSpace(fields["host"].Text())
		updated.RemoteHost = strings.TrimSpace(fields["remoteHost"].Text())
		updated.RemotePort = remotePort
		updated.LocalAddress = strings.TrimSpace(fields["localAddress"].Text())
		updated.LocalPort = localPort
		updated.WebProtocol = model.TunnelProtocolHTTP
		if protocol.CurrentIndex() == 1 {
			updated.WebProtocol = model.TunnelProtocolHTTPS
		}
		if validationErr := updated.Validate(); validationErr != nil {
			shell.SetValidation(validationErr.Error(), tunnelValidationField(fields, validationErr))
			return
		}
		accepted = true
		shell.SetValidation("", nil)
		shell.Accept()
	})
	if shell.Run() != walk.DlgCmdOK || !accepted {
		return model.TunnelDefinition{}, false, nil
	}
	return updated, true, nil
}

func addDialogField(parent walk.Container, labelText, value string) (*walk.LineEdit, error) {
	row, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	layout := walk.NewHBoxLayout()
	layout.SetMargins(walk.Margins{})
	layout.SetSpacing(8)
	if err := row.SetLayout(layout); err != nil {
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
	_ = layout.SetStretchFactor(input, 1)
	return input, nil
}

func parseDialogPort(name, value string) (uint16, error) {
	port, err := strconv.ParseUint(strings.TrimSpace(value), 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return uint16(port), nil
}

func tunnelValidationField(fields map[string]*walk.LineEdit, err error) walk.Widget {
	var validation *model.ValidationError
	if !errors.As(err, &validation) {
		return nil
	}
	return map[string]*walk.LineEdit{
		"hostAlias": fields["host"], "remoteHost": fields["remoteHost"],
		"remotePort": fields["remotePort"], "localAddress": fields["localAddress"],
		"localPort": fields["localPort"],
	}[validation.Field]
}
