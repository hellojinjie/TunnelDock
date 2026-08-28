package ui

import (
	"fmt"

	"github.com/tailscale/walk"
)

type TunnelListView struct {
	*walk.Composite
	env         *UIEnvironment
	callbacks   TunnelRowCallbacks
	emptyLabel  *walk.Label
	scroll      *walk.ScrollView
	container   *walk.Composite
	rows        map[string]*TunnelRowWidget
	order       []string
	busy        map[string]bool
	emptyText   string
	unsubscribe func()
}

func NewTunnelListView(parent walk.Container, env *UIEnvironment, callbacks TunnelRowCallbacks) (*TunnelListView, error) {
	root, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	view := &TunnelListView{
		Composite: root, env: env, callbacks: callbacks, rows: make(map[string]*TunnelRowWidget),
		busy: make(map[string]bool), emptyText: "No tunnels",
	}
	fail := func(cause error) (*TunnelListView, error) {
		root.Dispose()
		return nil, cause
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{})
	layout.SetSpacing(0)
	if err := root.SetLayout(layout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(root.DPI())
	if err != nil {
		return fail(err)
	}
	root.SetBackground(resources.WindowBrush)
	view.emptyLabel, err = walk.NewLabel(root)
	if err != nil {
		return fail(err)
	}
	_ = view.emptyLabel.SetText(view.emptyText)
	view.emptyLabel.SetFont(resources.BodyFont)
	view.scroll, err = walk.NewScrollView(root)
	if err != nil {
		return fail(err)
	}
	view.scroll.SetScrollbars(false, true)
	scrollLayout := walk.NewVBoxLayout()
	scrollLayout.SetMargins(walk.Margins{})
	scrollLayout.SetSpacing(6)
	if err := view.scroll.SetLayout(scrollLayout); err != nil {
		return fail(err)
	}
	view.container, err = newRowContainer(view.scroll)
	if err != nil {
		return fail(err)
	}
	view.emptyLabel.SetVisible(true)
	view.scroll.SetVisible(false)
	view.unsubscribe = env.Subscribe(func(Appearance) {
		if refreshed, resourceErr := env.Resources(view.DPI()); resourceErr == nil {
			view.SetBackground(refreshed.WindowBrush)
			for _, row := range view.rows {
				_ = row.updateHeight()
				row.Invalidate()
			}
		}
	})
	return view, nil
}

func (v *TunnelListView) SetRows(presentations []TunnelRowPresentation) error {
	next := make([]string, len(presentations))
	byID := make(map[string]TunnelRowPresentation, len(presentations))
	for index, presentation := range presentations {
		next[index] = presentation.ID
		byID[presentation.ID] = presentation
	}
	focusedID := ""
	for id, row := range v.rows {
		if row.Focused() {
			focusedID = id
			break
		}
	}
	for _, operation := range ReconcileRows(v.order, next) {
		switch operation.Kind {
		case ReconcileRemove:
			if row := v.rows[operation.ID]; row != nil {
				row.Dispose()
				delete(v.rows, operation.ID)
				delete(v.busy, operation.ID)
			}
		case ReconcileInsert:
			row, err := NewTunnelRowWidget(v.container, v.env, byID[operation.ID], v.callbacks)
			if err != nil {
				return fmt.Errorf("create tunnel row %q: %w", operation.ID, err)
			}
			row.SetBusy(v.busy[operation.ID])
			v.rows[operation.ID] = row
			children := v.container.Children()
			if at := children.Index(row); at != operation.To {
				_ = children.Remove(row)
				if err := children.Insert(operation.To, row); err != nil {
					return err
				}
			}
		case ReconcileMove:
			row := v.rows[operation.ID]
			children := v.container.Children()
			_ = children.Remove(row)
			if err := children.Insert(operation.To, row); err != nil {
				return err
			}
			row.SetPresentation(byID[operation.ID])
		case ReconcileKeep:
			v.rows[operation.ID].SetPresentation(byID[operation.ID])
		}
	}
	v.order = next
	hasRows := len(next) > 0
	v.emptyLabel.SetVisible(!hasRows)
	v.scroll.SetVisible(hasRows)
	if focusedID != "" {
		if row := v.rows[focusedID]; row != nil {
			_ = row.SetFocus()
		}
	}
	return nil
}

func (v *TunnelListView) SetBusy(runtimeID string, busy bool) {
	if busy {
		v.busy[runtimeID] = true
	} else {
		delete(v.busy, runtimeID)
	}
	if row := v.rows[runtimeID]; row != nil {
		row.SetBusy(busy)
	}
}

func (v *TunnelListView) SetEmptyText(text string) {
	if text == "" {
		text = "No tunnels"
	}
	v.emptyText = text
	_ = v.emptyLabel.SetText(text)
}

func (v *TunnelListView) Dispose() {
	if v.unsubscribe != nil {
		v.unsubscribe()
		v.unsubscribe = nil
	}
	v.Composite.Dispose()
}
