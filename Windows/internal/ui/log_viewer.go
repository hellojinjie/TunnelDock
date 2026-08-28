package ui

import (
	"context"
	"strings"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
)

func LogText(lines []string) string { return strings.Join(lines, "\r\n") }

func ShowTunnelLog(owner walk.Form, manager *tunnel.Manager, runtimeID string) error {
	window, err := walk.NewMainWindow()
	if err != nil {
		return err
	}
	if err := window.SetTitle("Tunnel Log"); err != nil {
		return err
	}
	if err := window.SetSize(walk.Size{Width: 760, Height: 480}); err != nil {
		return err
	}
	text, err := walk.NewTextEdit(window)
	if err != nil {
		return err
	}
	if err := text.SetReadOnly(true); err != nil {
		return err
	}
	if err := text.SetText("Loading log..."); err != nil {
		return err
	}
	if err := ApplyStandardTextScale(window); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	window.Closing().Attach(func(_ *bool, _ walk.CloseReason) { cancel() })
	window.Show()
	go refreshLog(ctx, manager, runtimeID, text)
	return nil
}

func refreshLog(ctx context.Context, manager *tunnel.Manager, runtimeID string, text *walk.TextEdit) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		if runtime, exists := manager.Snapshot(runtimeID); exists {
			value := LogText(runtime.LogLines)
			walk.App().Synchronize(func() { _ = text.SetText(value) })
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
