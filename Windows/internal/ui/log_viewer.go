package ui

import (
	"context"
	"strings"
	"time"

	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
)

func LogText(lines []string) string { return strings.Join(lines, "\r\n") }

func ShowTunnelLog(owner walk.Form, env *UIEnvironment, manager *tunnel.Manager, runtimeID string) error {
	_ = owner
	window, err := walk.NewMainWindow()
	if err != nil {
		return err
	}
	fail := func(cause error) error {
		window.Dispose()
		return cause
	}
	if err := window.SetTitle("Tunnel Log"); err != nil {
		return fail(err)
	}
	if err := window.SetSize(walk.Size{Width: 780, Height: 520}); err != nil {
		return fail(err)
	}
	if err := window.SetMinMaxSize(walk.Size{Width: 560, Height: 360}, walk.Size{}); err != nil {
		return fail(err)
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 18, VNear: 16, HFar: 18, VFar: 16})
	layout.SetSpacing(8)
	if err := window.SetLayout(layout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(window.DPI())
	if err != nil {
		return fail(err)
	}
	window.SetBackground(resources.WindowBrush)
	title, err := walk.NewLabel(window)
	if err != nil {
		return fail(err)
	}
	_ = title.SetText("Tunnel Log")
	title.SetFont(resources.TitleFont)
	state, err := walk.NewLabel(window)
	if err != nil {
		return fail(err)
	}
	state.SetFont(resources.CaptionFont)
	card, err := NewCard(window, env)
	if err != nil {
		return fail(err)
	}
	cardLayout := walk.NewVBoxLayout()
	cardLayout.SetMargins(walk.Margins{HNear: 12, VNear: 12, HFar: 12, VFar: 12})
	if err := card.Content.SetLayout(cardLayout); err != nil {
		card.Dispose()
		return fail(err)
	}
	text, err := walk.NewTextEdit(card.Content)
	if err != nil {
		card.Dispose()
		return fail(err)
	}
	if err := text.SetReadOnly(true); err != nil {
		card.Dispose()
		return fail(err)
	}
	text.SetFont(resources.MonoFont)
	_ = text.SetText("Loading log...")
	_ = layout.SetStretchFactor(card, 1)
	_ = cardLayout.SetStretchFactor(text, 1)
	if err := env.ApplyNativeFont(window, window.DPI()); err != nil {
		card.Dispose()
		return fail(err)
	}
	title.SetFont(resources.TitleFont)
	text.SetFont(resources.MonoFont)
	ctx, cancel := context.WithCancel(context.Background())
	unsubscribe := env.Subscribe(func(Appearance) {
		if refreshed, resourceErr := env.Resources(window.DPI()); resourceErr == nil {
			window.SetBackground(refreshed.WindowBrush)
			text.SetFont(refreshed.MonoFont)
			window.Invalidate()
		}
	})
	closing := false
	window.Closing().Attach(func(_ *bool, _ walk.CloseReason) {
		if closing {
			return
		}
		closing = true
		cancel()
		unsubscribe()
		card.Dispose()
	})
	window.Show()
	go refreshLog(ctx, manager, runtimeID, title, state, text)
	return nil
}

func refreshLog(ctx context.Context, manager *tunnel.Manager, runtimeID string, title, state *walk.Label, text *walk.TextEdit) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	last := ""
	for {
		if runtime, exists := manager.Snapshot(runtimeID); exists {
			value := LogText(runtime.LogLines)
			walk.App().Synchronize(func() {
				if ctx.Err() != nil {
					return
				}
				_ = title.SetText(runtime.DisplayName())
				_ = state.SetText(tunnelStateText(runtime.State))
				if value == last {
					return
				}
				selectionStart, selectionEnd := text.TextSelection()
				followTail := selectionEnd >= text.TextLength()
				_ = text.SetText(value)
				if followTail {
					text.SetTextSelection(text.TextLength(), text.TextLength())
					text.ScrollToCaret()
				} else {
					text.SetTextSelection(selectionStart, selectionEnd)
				}
				last = value
			})
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
