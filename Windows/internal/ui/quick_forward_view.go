package ui

import (
	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type QuickForwardPresentation struct {
	ConnectText      string
	ConnectEnabled   bool
	AdvancedExpanded bool
	FocusField       app.FocusTarget
}

func PresentQuickForward(quick *app.QuickForward, busy bool) QuickForwardPresentation {
	state := QuickForwardPresentation{
		ConnectText:      "Connect",
		ConnectEnabled:   quick != nil && quick.HasRemotePort() && !busy,
		AdvancedExpanded: quick != nil && quick.AdvancedExpanded,
	}
	if quick != nil {
		state.FocusField = quick.Focus
	}
	if busy {
		state.ConnectText = "Connecting..."
	}
	return state
}

type QuickForwardView struct {
	*Card
	env           *UIEnvironment
	quick         *app.QuickForward
	onConnect     func()
	title         *walk.Label
	remotePort    *walk.LineEdit
	connect       *walk.PushButton
	disclosure    *quickDisclosure
	advanced      *walk.Composite
	localPort     *walk.LineEdit
	remoteHost    *walk.LineEdit
	localAddress  *walk.LineEdit
	protocol      *walk.ComboBox
	validation    *walk.Label
	busy          bool
	hostAvailable bool
	syncing       bool
}

func NewQuickForwardView(parent walk.Container, env *UIEnvironment, quick *app.QuickForward, onConnect func()) (*QuickForwardView, error) {
	card, err := NewCard(parent, env)
	if err != nil {
		return nil, err
	}
	view := &QuickForwardView{Card: card, env: env, quick: quick, onConnect: onConnect, hostAvailable: true}
	fail := func(cause error) (*QuickForwardView, error) {
		card.Dispose()
		return nil, cause
	}
	resources, err := env.Resources(card.DPI())
	if err != nil {
		return fail(err)
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 16, VNear: 14, HFar: 16, VFar: 14})
	layout.SetSpacing(10)
	if err := card.Content.SetLayout(layout); err != nil {
		return fail(err)
	}
	view.title, err = walk.NewLabel(card.Content)
	if err != nil {
		return fail(err)
	}
	_ = view.title.SetText("Quick Forward")
	view.title.SetFont(resources.MediumFont)
	primary, err := walk.NewComposite(card.Content)
	if err != nil {
		return fail(err)
	}
	primaryLayout := walk.NewHBoxLayout()
	primaryLayout.SetMargins(walk.Margins{})
	primaryLayout.SetSpacing(8)
	if err := primary.SetLayout(primaryLayout); err != nil {
		return fail(err)
	}
	view.remotePort, err = walk.NewLineEdit(primary)
	if err != nil {
		return fail(err)
	}
	_ = view.remotePort.SetCueBanner("Remote Port")
	view.remotePort.TextChanged().Attach(func() {
		if view.syncing {
			return
		}
		view.quick.SetRemotePort(view.remotePort.Text())
		view.syncFromModel()
	})
	view.connect, err = walk.NewPushButton(primary)
	if err != nil {
		return fail(err)
	}
	_ = view.connect.SetText("Connect")
	view.connect.Clicked().Attach(func() {
		if view.onConnect != nil && view.connect.Enabled() {
			view.onConnect()
		}
	})
	view.disclosure, err = newQuickDisclosure(card.Content, env, func() {
		view.quick.AdvancedExpanded = !view.quick.AdvancedExpanded
		view.syncFromModel()
	})
	if err != nil {
		return fail(err)
	}
	view.advanced, err = walk.NewComposite(card.Content)
	if err != nil {
		return fail(err)
	}
	advancedLayout := walk.NewGridLayout()
	advancedLayout.SetMargins(walk.Margins{})
	advancedLayout.SetSpacing(8)
	_ = advancedLayout.SetColumnStretchFactor(1, 1)
	if err := view.advanced.SetLayout(advancedLayout); err != nil {
		return fail(err)
	}
	var label *walk.Label
	if label, err = addQuickLabel(view.advanced, "Local Port", resources); err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(label, walk.Rectangle{X: 0, Y: 0, Width: 1, Height: 1})
	view.localPort, err = walk.NewLineEdit(view.advanced)
	if err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(view.localPort, walk.Rectangle{X: 1, Y: 0, Width: 1, Height: 1})
	view.localPort.TextChanged().Attach(func() {
		if !view.syncing {
			view.quick.SetLocalPort(view.localPort.Text())
		}
	})
	if label, err = addQuickLabel(view.advanced, "Remote Host", resources); err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(label, walk.Rectangle{X: 0, Y: 1, Width: 1, Height: 1})
	view.remoteHost, err = walk.NewLineEdit(view.advanced)
	if err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(view.remoteHost, walk.Rectangle{X: 1, Y: 1, Width: 1, Height: 1})
	view.remoteHost.TextChanged().Attach(func() {
		if !view.syncing {
			view.quick.RemoteHost = view.remoteHost.Text()
		}
	})
	if label, err = addQuickLabel(view.advanced, "Local Address", resources); err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(label, walk.Rectangle{X: 0, Y: 2, Width: 1, Height: 1})
	view.localAddress, err = walk.NewLineEdit(view.advanced)
	if err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(view.localAddress, walk.Rectangle{X: 1, Y: 2, Width: 1, Height: 1})
	view.localAddress.TextChanged().Attach(func() {
		if !view.syncing {
			view.quick.LocalAddress = view.localAddress.Text()
		}
	})
	if label, err = addQuickLabel(view.advanced, "Browser Protocol", resources); err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(label, walk.Rectangle{X: 0, Y: 3, Width: 1, Height: 1})
	view.protocol, err = walk.NewComboBox(view.advanced)
	if err != nil {
		return fail(err)
	}
	if err := view.protocol.SetModel([]string{"http", "https"}); err != nil {
		return fail(err)
	}
	_ = advancedLayout.SetRange(view.protocol, walk.Rectangle{X: 1, Y: 3, Width: 1, Height: 1})
	view.protocol.CurrentIndexChanged().Attach(func() {
		if view.syncing {
			return
		}
		if view.protocol.CurrentIndex() == 1 {
			view.quick.WebProtocol = model.TunnelProtocolHTTPS
		} else {
			view.quick.WebProtocol = model.TunnelProtocolHTTP
		}
	})
	view.validation, err = walk.NewLabel(card.Content)
	if err != nil {
		return fail(err)
	}
	view.validation.SetTextColor(resources.Palette.Failure)
	setChildVisible(view.validation, false)
	view.syncFromModel()
	return view, nil
}

func addQuickLabel(parent walk.Container, text string, resources *UIResources) (*walk.Label, error) {
	label, err := walk.NewLabel(parent)
	if err != nil {
		return nil, err
	}
	_ = label.SetText(text)
	label.SetFont(resources.BodyFont)
	return label, nil
}

func (v *QuickForwardView) SetHostAvailable(available bool) {
	v.hostAvailable = available
	v.syncFromModel()
}

func (v *QuickForwardView) SetBusy(busy bool) {
	v.busy = busy
	v.syncFromModel()
}

func (v *QuickForwardView) SetValidation(message string) {
	_ = v.validation.SetText(message)
	setChildVisible(v.validation, message != "")
}

func (v *QuickForwardView) ApplyModelFocus() {
	if v.quick.Focus == app.FocusLocalPort {
		v.quick.AdvancedExpanded = true
		v.syncFromModel()
		_ = v.localPort.SetFocus()
		v.quick.Focus = app.FocusNone
	}
}

func (v *QuickForwardView) syncFromModel() {
	if v.quick == nil {
		return
	}
	v.syncing = true
	defer func() { v.syncing = false }()
	if v.remotePort.Text() != v.quick.RemotePort {
		_ = v.remotePort.SetText(v.quick.RemotePort)
	}
	if v.localPort.Text() != v.quick.LocalPort {
		_ = v.localPort.SetText(v.quick.LocalPort)
	}
	if v.remoteHost.Text() != v.quick.RemoteHost {
		_ = v.remoteHost.SetText(v.quick.RemoteHost)
	}
	if v.localAddress.Text() != v.quick.LocalAddress {
		_ = v.localAddress.SetText(v.quick.LocalAddress)
	}
	protocolIndex := 0
	if v.quick.WebProtocol == model.TunnelProtocolHTTPS {
		protocolIndex = 1
	}
	if v.protocol.CurrentIndex() != protocolIndex {
		_ = v.protocol.SetCurrentIndex(protocolIndex)
	}
	state := PresentQuickForward(v.quick, v.busy)
	visibility := detailVisibilityFor(false, 0, state.AdvancedExpanded)
	setChildVisible(v.advanced, visibility.Advanced)
	v.disclosure.SetExpanded(state.AdvancedExpanded)
	_ = v.connect.SetText(state.ConnectText)
	v.connect.SetEnabled(state.ConnectEnabled && v.hostAvailable && v.onConnect != nil)
}

type quickDisclosure struct {
	*walk.CustomWidget
	env      *UIEnvironment
	expanded bool
	pressed  bool
	toggle   func()
}

func newQuickDisclosure(parent walk.Container, env *UIEnvironment, toggle func()) (*quickDisclosure, error) {
	button := &quickDisclosure{env: env, toggle: toggle}
	custom, err := walk.NewCustomWidgetPixels(parent, uint(win.WS_TABSTOP), func(canvas *walk.Canvas, bounds walk.Rectangle) error {
		return button.paint(canvas, bounds)
	})
	if err != nil {
		return nil, err
	}
	button.CustomWidget = custom
	button.SetPaintMode(walk.PaintBuffered)
	button.SetInvalidatesOnResize(true)
	resources, err := env.Resources(button.DPI())
	if err != nil {
		button.Dispose()
		return nil, err
	}
	height := resources.Metrics.ActionHeight
	if err := button.SetMinMaxSizePixels(walk.Size{Height: height}, walk.Size{Height: height}); err != nil {
		button.Dispose()
		return nil, err
	}
	button.MouseDown().Attach(func(_, _ int, mouseButton walk.MouseButton) {
		if mouseButton == walk.LeftButton {
			button.pressed = true
			_ = button.SetFocus()
			button.Invalidate()
		}
	})
	button.MouseUp().Attach(func(_, _ int, mouseButton walk.MouseButton) {
		if mouseButton == walk.LeftButton && button.pressed {
			button.pressed = false
			button.Invalidate()
			if button.toggle != nil {
				button.toggle()
			}
		}
	})
	button.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn || key == walk.KeySpace {
			if button.toggle != nil {
				button.toggle()
			}
		}
	})
	button.FocusedChanged().Attach(func() { button.Invalidate() })
	button.SetToolTipText("Show or hide advanced forwarding options")
	return button, nil
}

func (b *quickDisclosure) SetExpanded(expanded bool) {
	if b.expanded != expanded {
		b.expanded = expanded
		b.Invalidate()
	}
}

func (b *quickDisclosure) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	resources, err := b.env.Resources(b.DPI())
	if err != nil {
		return err
	}
	bounds := b.ClientBoundsPixels()
	if err := canvas.FillRectanglePixels(resources.SurfaceBrush, bounds); err != nil {
		return err
	}
	if b.pressed {
		if err := FillRoundedSurface(canvas, resources.HoverBrush, bounds, resources.Metrics.RowRadius); err != nil {
			return err
		}
	}
	iconSize := resources.Metrics.IconSize
	iconBounds := walk.Rectangle{X: 2, Y: (bounds.Height - iconSize) / 2, Width: iconSize, Height: iconSize}
	icon := IconChevronRight
	if b.expanded {
		icon = IconChevronDown
	}
	if err := DrawIcon(canvas, resources, icon, iconBounds, resources.Palette.SecondaryText); err != nil {
		return err
	}
	textBounds := walk.Rectangle{X: iconBounds.X + iconBounds.Width + 4, Y: 0, Width: max(1, bounds.Width-iconBounds.Width-8), Height: bounds.Height}
	if err := DrawTextEllipsized(canvas, "Advanced", resources.BodyFont, resources.Palette.PrimaryText, textBounds); err != nil {
		return err
	}
	if b.Focused() {
		return DrawFocusRing(canvas, resources, insetRect(bounds, 1), resources.Metrics.RowRadius)
	}
	return nil
}
