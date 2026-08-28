package ui

import (
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

type TunnelMenuItem uint8

const (
	TunnelMenuLog TunnelMenuItem = iota
	TunnelMenuSave
	TunnelMenuRename
	TunnelMenuEdit
	TunnelMenuDelete
)

type TunnelMenuEntry struct {
	Item        TunnelMenuItem
	Text        string
	Enabled     bool
	Destructive bool
}

type TunnelMenuModel []TunnelMenuEntry

func (m TunnelMenuModel) Contains(item TunnelMenuItem) bool {
	for _, entry := range m {
		if entry.Item == item {
			return true
		}
	}
	return false
}

func (m TunnelMenuModel) Enabled(item TunnelMenuItem) bool {
	for _, entry := range m {
		if entry.Item == item {
			return entry.Enabled
		}
	}
	return false
}

func MoreMenuItems(row TunnelRowPresentation) TunnelMenuModel {
	items := TunnelMenuModel{{Item: TunnelMenuLog, Text: "View Log", Enabled: true}}
	if row.Temporary {
		return append(items, TunnelMenuEntry{Item: TunnelMenuSave, Text: "Save Tunnel...", Enabled: true})
	}
	running := row.State == model.StateConnecting || row.State == model.StateConnected || row.State == model.StateReconnecting
	return append(items,
		TunnelMenuEntry{Item: TunnelMenuRename, Text: "Rename...", Enabled: true},
		TunnelMenuEntry{Item: TunnelMenuEdit, Text: "Edit...", Enabled: !running},
		TunnelMenuEntry{Item: TunnelMenuDelete, Text: "Delete...", Enabled: !running, Destructive: true},
	)
}

type TunnelRowCallbacks struct {
	Primary     func(string, TunnelRowAction)
	OpenBrowser func(string)
	ViewLog     func(string)
	Save        func(string)
	Rename      func(string)
	Edit        func(string)
	Delete      func(string)
}

type TunnelRowWidget struct {
	*walk.CustomWidget
	env          *UIEnvironment
	presentation TunnelRowPresentation
	callbacks    TunnelRowCallbacks
	busy         bool
	hovered      TunnelRowAction
	pressed      TunnelRowAction
	focusAction  TunnelRowAction
}

func NewTunnelRowWidget(parent walk.Container, env *UIEnvironment, row TunnelRowPresentation, callbacks TunnelRowCallbacks) (*TunnelRowWidget, error) {
	widget := &TunnelRowWidget{env: env, presentation: row, callbacks: callbacks}
	custom, err := walk.NewCustomWidgetPixels(parent, uint(win.WS_TABSTOP), func(canvas *walk.Canvas, bounds walk.Rectangle) error {
		return widget.paint(canvas, bounds)
	})
	if err != nil {
		return nil, err
	}
	widget.CustomWidget = custom
	widget.SetPaintMode(walk.PaintBuffered)
	widget.SetInvalidatesOnResize(true)
	widget.focusAction = widget.firstAction()
	if err := widget.updateHeight(); err != nil {
		widget.Dispose()
		return nil, err
	}
	widget.SetToolTipText(widget.toolTip())
	widget.MouseMove().Attach(widget.onMouseMove)
	widget.MouseDown().Attach(widget.onMouseDown)
	widget.MouseUp().Attach(widget.onMouseUp)
	widget.KeyDown().Attach(widget.onKeyDown)
	widget.FocusedChanged().Attach(func() { widget.Invalidate() })
	return widget, nil
}

func (w *TunnelRowWidget) SetPresentation(row TunnelRowPresentation) {
	w.presentation = row
	w.focusAction = w.normalizedFocusAction(w.focusAction)
	w.SetToolTipText(w.toolTip())
	_ = w.updateHeight()
	w.Invalidate()
}

func (w *TunnelRowWidget) SetBusy(busy bool) {
	if w.busy == busy {
		return
	}
	w.busy = busy
	w.Invalidate()
}

func (w *TunnelRowWidget) updateHeight() error {
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return err
	}
	height := TunnelRowHeight(w.presentation, resources.Metrics)
	return w.SetMinMaxSize(walk.Size{Height: height}, walk.Size{Height: height})
}

func (w *TunnelRowWidget) toolTip() string {
	text := fmt.Sprintf("%s\n%s", w.presentation.Name, w.presentation.Forward)
	if w.presentation.ErrorText != "" {
		text += "\n" + w.presentation.ErrorText
	}
	return text
}

func (w *TunnelRowWidget) availableActions() []TunnelRowAction {
	actions := make([]TunnelRowAction, 0, 3)
	if w.presentation.ShowBrowser {
		actions = append(actions, TunnelRowOpenBrowser)
	}
	if w.presentation.PrimaryAction != TunnelRowNoAction {
		actions = append(actions, w.presentation.PrimaryAction)
	}
	if w.presentation.ShowMore {
		actions = append(actions, TunnelRowMore)
	}
	return actions
}

func (w *TunnelRowWidget) firstAction() TunnelRowAction {
	actions := w.availableActions()
	if len(actions) == 0 {
		return TunnelRowNoAction
	}
	return actions[0]
}

func (w *TunnelRowWidget) normalizedFocusAction(action TunnelRowAction) TunnelRowAction {
	for _, candidate := range w.availableActions() {
		if candidate == action {
			return action
		}
	}
	return w.firstAction()
}

func (w *TunnelRowWidget) onMouseMove(x, y int, _ walk.MouseButton) {
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return
	}
	action := LayoutTunnelRow(w.ClientBoundsPixels().Width, w.ClientBoundsPixels().Height, resources.Metrics, w.presentation).HitTest(x, y)
	if action != w.hovered {
		w.hovered = action
		w.Invalidate()
	}
}

func (w *TunnelRowWidget) onMouseDown(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton || w.busy {
		return
	}
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return
	}
	w.pressed = LayoutTunnelRow(w.ClientBoundsPixels().Width, w.ClientBoundsPixels().Height, resources.Metrics, w.presentation).HitTest(x, y)
	if w.pressed != TunnelRowNoAction {
		w.focusAction = w.pressed
		_ = w.SetFocus()
		w.Invalidate()
	}
}

func (w *TunnelRowWidget) onMouseUp(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton || w.pressed == TunnelRowNoAction {
		return
	}
	pressed := w.pressed
	w.pressed = TunnelRowNoAction
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return
	}
	current := LayoutTunnelRow(w.ClientBoundsPixels().Width, w.ClientBoundsPixels().Height, resources.Metrics, w.presentation).HitTest(x, y)
	w.Invalidate()
	if current == pressed {
		w.invoke(pressed)
	}
}

func (w *TunnelRowWidget) onKeyDown(key walk.Key) {
	actions := w.availableActions()
	if len(actions) == 0 {
		return
	}
	index := 0
	for i, action := range actions {
		if action == w.focusAction {
			index = i
			break
		}
	}
	switch key {
	case walk.KeyLeft:
		w.focusAction = actions[(index+len(actions)-1)%len(actions)]
		w.Invalidate()
	case walk.KeyRight:
		w.focusAction = actions[(index+1)%len(actions)]
		w.Invalidate()
	case walk.KeyReturn, walk.KeySpace:
		if !w.busy {
			w.invoke(w.focusAction)
		}
	}
}

func (w *TunnelRowWidget) invoke(action TunnelRowAction) {
	switch action {
	case TunnelRowOpenBrowser:
		if w.callbacks.OpenBrowser != nil {
			w.callbacks.OpenBrowser(w.presentation.ID)
		}
	case TunnelRowConnect, TunnelRowDisconnect:
		if w.callbacks.Primary != nil {
			w.callbacks.Primary(w.presentation.ID, action)
		}
	case TunnelRowMore:
		w.showMoreMenu()
	}
}

func (w *TunnelRowWidget) showMoreMenu() {
	menu, err := walk.NewMenu()
	if err != nil {
		return
	}
	defer menu.Dispose()
	for _, entry := range MoreMenuItems(w.presentation) {
		entry := entry
		action := walk.NewAction()
		_ = action.SetText(entry.Text)
		_ = action.SetEnabled(entry.Enabled && !w.busy)
		action.Triggered().Attach(func() { w.invokeMenu(entry.Item) })
		if err := menu.Actions().Add(action); err != nil {
			action.Dispose()
			continue
		}
	}
	w.SetContextMenu(menu)
	win.SendMessage(w.Handle(), win.WM_CONTEXTMENU, uintptr(w.Handle()), ^uintptr(0))
	w.SetContextMenu(nil)
}

func (w *TunnelRowWidget) invokeMenu(item TunnelMenuItem) {
	callback := map[TunnelMenuItem]func(string){
		TunnelMenuLog: w.callbacks.ViewLog, TunnelMenuSave: w.callbacks.Save,
		TunnelMenuRename: w.callbacks.Rename, TunnelMenuEdit: w.callbacks.Edit,
		TunnelMenuDelete: w.callbacks.Delete,
	}[item]
	if callback != nil {
		callback(w.presentation.ID)
	}
}

func (w *TunnelRowWidget) paint(canvas *walk.Canvas, _ walk.Rectangle) error {
	resources, err := w.env.Resources(w.DPI())
	if err != nil {
		return err
	}
	bounds := w.ClientBoundsPixels()
	if err := canvas.FillRectanglePixels(resources.WindowBrush, bounds); err != nil {
		return err
	}
	rowBounds := insetRect(bounds, 2)
	if err := FillRoundedSurface(canvas, resources.SurfaceBrush, rowBounds, resources.Metrics.RowRadius); err != nil {
		return err
	}
	if err := canvas.DrawRoundedRectanglePixels(resources.BorderPen, rowBounds, walk.Size{Width: resources.Metrics.RowRadius * 2, Height: resources.Metrics.RowRadius * 2}); err != nil {
		return err
	}
	layout := LayoutTunnelRow(bounds.Width, bounds.Height, resources.Metrics, w.presentation)
	stateIcon, stateColor := tunnelStateAppearance(w.presentation.State, resources.Palette)
	if err := DrawIcon(canvas, resources, stateIcon, toWalkRect(layout.StateIcon), stateColor); err != nil {
		return err
	}
	lineHeight := max(18, layout.Text.Height/3)
	nameBounds := walk.Rectangle{X: layout.Text.X, Y: layout.Text.Y, Width: layout.Text.Width, Height: lineHeight}
	if err := DrawTextEllipsized(canvas, w.presentation.Name, resources.MediumFont, resources.Palette.PrimaryText, nameBounds); err != nil {
		return err
	}
	forwardBounds := walk.Rectangle{X: layout.Text.X, Y: layout.Text.Y + lineHeight, Width: layout.Text.Width, Height: lineHeight}
	if err := DrawTextEllipsized(canvas, w.presentation.Forward, resources.CaptionFont, resources.Palette.SecondaryText, forwardBounds); err != nil {
		return err
	}
	if w.presentation.ErrorText != "" {
		errorBounds := walk.Rectangle{X: layout.Text.X, Y: layout.Text.Y + lineHeight*2, Width: layout.Text.Width, Height: lineHeight}
		if err := DrawTextEllipsized(canvas, w.presentation.ErrorText, resources.CaptionFont, resources.Palette.Failure, errorBounds); err != nil {
			return err
		}
	}
	if err := DrawTextEllipsized(canvas, w.presentation.StateText, resources.CaptionFont, stateColor, toWalkRect(layout.StateLabel)); err != nil {
		return err
	}
	if w.presentation.ShowBrowser {
		if err := w.paintIconAction(canvas, resources, layout.Browser, TunnelRowOpenBrowser, IconBrowser); err != nil {
			return err
		}
	}
	if w.presentation.PrimaryAction != TunnelRowNoAction {
		text := w.presentation.PrimaryText
		if w.busy {
			text = "Working..."
		}
		if err := w.paintTextAction(canvas, resources, layout.Primary, w.presentation.PrimaryAction, text); err != nil {
			return err
		}
	}
	if w.presentation.ShowMore {
		if err := w.paintIconAction(canvas, resources, layout.More, TunnelRowMore, IconMore); err != nil {
			return err
		}
	}
	if w.Focused() && w.focusAction != TunnelRowNoAction {
		return DrawFocusRing(canvas, resources, insetRect(toWalkRect(actionRect(layout, w.focusAction)), 1), resources.Metrics.RowRadius)
	}
	return nil
}

func (w *TunnelRowWidget) paintIconAction(canvas *walk.Canvas, resources *UIResources, bounds Rect, action TunnelRowAction, icon IconKind) error {
	if err := w.paintActionSurface(canvas, resources, bounds, action); err != nil {
		return err
	}
	iconBounds := insetRect(toWalkRect(bounds), max(5, resources.Metrics.IconSize/3))
	color := resources.Palette.SecondaryText
	if w.busy {
		color = resources.Palette.DisabledText
	}
	return DrawIcon(canvas, resources, icon, iconBounds, color)
}

func (w *TunnelRowWidget) paintTextAction(canvas *walk.Canvas, resources *UIResources, bounds Rect, action TunnelRowAction, text string) error {
	if err := w.paintActionSurface(canvas, resources, bounds, action); err != nil {
		return err
	}
	color := resources.Palette.Accent
	if w.busy {
		color = resources.Palette.DisabledText
	}
	return DrawTextEllipsized(canvas, text, resources.MediumFont, color, toWalkRect(bounds))
}

func (w *TunnelRowWidget) paintActionSurface(canvas *walk.Canvas, resources *UIResources, bounds Rect, action TunnelRowAction) error {
	brush := resources.SurfaceBrush
	if w.pressed == action {
		brush = resources.SelectedBrush
	} else if w.hovered == action {
		brush = resources.HoverBrush
	}
	return FillRoundedSurface(canvas, brush, toWalkRect(bounds), resources.Metrics.RowRadius)
}

func tunnelStateAppearance(state model.TunnelState, palette Palette) (IconKind, walk.Color) {
	switch state {
	case model.StateConnecting, model.StateReconnecting:
		return IconConnecting, palette.Connecting
	case model.StateConnected:
		return IconConnected, palette.Success
	case model.StateFailed:
		return IconFailed, palette.Failure
	default:
		return IconDisconnected, palette.SecondaryText
	}
}

func actionRect(layout TunnelRowLayout, action TunnelRowAction) Rect {
	switch action {
	case TunnelRowOpenBrowser:
		return layout.Browser
	case TunnelRowConnect, TunnelRowDisconnect:
		return layout.Primary
	case TunnelRowMore:
		return layout.More
	default:
		return Rect{}
	}
}

func toWalkRect(bounds Rect) walk.Rectangle {
	return walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height}
}
