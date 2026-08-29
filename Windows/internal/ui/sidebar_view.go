package ui

import (
	"fmt"

	"github.com/tailscale/walk"
)

const allTunnelsPaneID = "__all_tunnels__"

type sidebarState struct {
	selected string
	ids      map[string]bool
}

func newSidebarState() *sidebarState {
	return &sidebarState{selected: allTunnelsPaneID, ids: make(map[string]bool)}
}

func (s *sidebarState) SetSelected(id string) {
	if id == "" {
		id = allTunnelsPaneID
	}
	s.selected = id
}

func (s *sidebarState) Selected() string { return s.selected }

func (s *sidebarState) Apply(rows []HostRowPresentation) {
	s.ids = make(map[string]bool, len(rows))
	for _, row := range rows {
		s.ids[row.ID] = true
	}
	if s.selected != allTunnelsPaneID && !s.ids[s.selected] {
		s.selected = allTunnelsPaneID
	}
}

type SidebarCallbacks struct {
	SelectPane func(string)
	Search     func(string)
	AddHost    func()
	EditConfig func()
	Refresh    func()
}

type SidebarView struct {
	*walk.Composite
	env              *UIEnvironment
	callbacks        SidebarCallbacks
	state            *sidebarState
	allRow           *HostRowWidget
	search           *walk.LineEdit
	scroll           *walk.ScrollView
	currentContainer *walk.Composite
	missingLabel     *walk.Label
	missingContainer *walk.Composite
	currentRows      map[string]*HostRowWidget
	missingRows      map[string]*HostRowWidget
	currentOrder     []string
	missingOrder     []string
	unsubscribe      func()
}

func NewSidebarView(parent walk.Container, env *UIEnvironment, callbacks SidebarCallbacks) (*SidebarView, error) {
	root, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	view := &SidebarView{
		Composite: root, env: env, callbacks: callbacks, state: newSidebarState(),
		currentRows: make(map[string]*HostRowWidget), missingRows: make(map[string]*HostRowWidget),
	}
	fail := func(cause error) (*SidebarView, error) {
		root.Dispose()
		return nil, cause
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 10, VNear: 10, HFar: 10, VFar: 10})
	layout.SetSpacing(6)
	if err := root.SetLayout(layout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(root.DPI())
	if err != nil {
		return fail(err)
	}
	root.SetBackground(resources.SidebarBrush)
	view.allRow, err = NewHostRowWidget(root, env, HostRowPresentation{ID: allTunnelsPaneID, Title: "All Tunnels"}, view.selectPane)
	if err != nil {
		return fail(err)
	}
	view.allRow.SetSelected(true)
	title, err := walk.NewLabel(root)
	if err != nil {
		return fail(err)
	}
	_ = title.SetText("SSH Hosts")
	title.SetFont(resources.CaptionFont)
	view.search, err = walk.NewLineEdit(root)
	if err != nil {
		return fail(err)
	}
	_ = view.search.SetCueBanner("Search SSH hosts")
	view.search.TextChanged().Attach(func() {
		if view.callbacks.Search != nil {
			view.callbacks.Search(view.search.Text())
		}
	})
	toolbar, err := walk.NewComposite(root)
	if err != nil {
		return fail(err)
	}
	toolbarLayout := walk.NewHBoxLayout()
	toolbarLayout.SetMargins(walk.Margins{})
	toolbarLayout.SetSpacing(4)
	_ = toolbar.SetLayout(toolbarLayout)
	policy := defaultWindowLayoutPolicy()
	if err := toolbar.SetMinMaxSize(walk.Size{Height: policy.ToolbarHeight}, walk.Size{Height: policy.ToolbarHeight}); err != nil {
		return fail(err)
	}
	for _, action := range []struct {
		icon IconKind
		tip  string
		fn   func()
	}{
		{IconPlus, "Add SSH Host", callbacks.AddHost},
		{IconEdit, "Open SSH config", callbacks.EditConfig},
		{IconRefresh, "Refresh SSH config", callbacks.Refresh},
	} {
		_, buttonErr := NewIconButton(toolbar, env, iconButtonOnSidebar, action.icon, action.tip, action.fn)
		if buttonErr != nil {
			return fail(buttonErr)
		}
	}
	if _, err = walk.NewHSpacer(toolbar); err != nil {
		return fail(err)
	}
	view.scroll, err = walk.NewScrollView(root)
	if err != nil {
		return fail(err)
	}
	view.scroll.SetScrollbars(true, true)
	scrollLayout := walk.NewVBoxLayout()
	scrollLayout.SetMargins(walk.Margins{})
	scrollLayout.SetSpacing(2)
	if err := view.scroll.SetLayout(scrollLayout); err != nil {
		return fail(err)
	}
	view.currentContainer, err = newRowContainer(view.scroll)
	if err != nil {
		return fail(err)
	}
	view.missingLabel, err = walk.NewLabel(view.scroll)
	if err != nil {
		return fail(err)
	}
	_ = view.missingLabel.SetText("Missing Hosts")
	view.missingLabel.SetFont(resources.CaptionFont)
	view.missingContainer, err = newRowContainer(view.scroll)
	if err != nil {
		return fail(err)
	}
	setChildVisible(view.missingLabel, false)
	setChildVisible(view.missingContainer, false)
	view.unsubscribe = env.Subscribe(func(Appearance) {
		if refreshed, resourceErr := env.Resources(view.DPI()); resourceErr == nil {
			view.SetBackground(refreshed.SidebarBrush)
			view.allRow.Invalidate()
			for _, row := range view.currentRows {
				row.Invalidate()
			}
			for _, row := range view.missingRows {
				row.Invalidate()
			}
		}
	})
	return view, nil
}

func newRowContainer(parent walk.Container) (*walk.Composite, error) {
	container, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{})
	layout.SetSpacing(2)
	layout.SetAlignment(defaultWindowLayoutPolicy().RowAlignment)
	if err := container.SetLayout(layout); err != nil {
		container.Dispose()
		return nil, err
	}
	return container, nil
}

func (v *SidebarView) SetRows(current, missing []HostRowPresentation) error {
	all := make([]HostRowPresentation, 0, len(current)+len(missing))
	all = append(all, current...)
	all = append(all, missing...)
	v.state.Apply(all)
	if err := v.reconcileContainer(v.currentContainer, v.currentRows, &v.currentOrder, current); err != nil {
		return err
	}
	if err := v.reconcileContainer(v.missingContainer, v.missingRows, &v.missingOrder, missing); err != nil {
		return err
	}
	visible := len(missing) > 0
	setChildVisible(v.missingLabel, visible)
	setChildVisible(v.missingContainer, visible)
	v.applySelection()
	return nil
}

func (v *SidebarView) SetSelected(id string) {
	v.state.SetSelected(id)
	v.applySelection()
}

func (v *SidebarView) Selected() string   { return v.state.Selected() }
func (v *SidebarView) SearchText() string { return v.search.Text() }

func (v *SidebarView) selectPane(id string) {
	v.SetSelected(id)
	if v.callbacks.SelectPane != nil {
		v.callbacks.SelectPane(id)
	}
}

func (v *SidebarView) applySelection() {
	selected := v.state.Selected()
	v.allRow.SetSelected(selected == allTunnelsPaneID)
	for id, row := range v.currentRows {
		row.SetSelected(id == selected)
	}
	for id, row := range v.missingRows {
		row.SetSelected(id == selected)
	}
}

func (v *SidebarView) reconcileContainer(container *walk.Composite, widgets map[string]*HostRowWidget, order *[]string, rows []HostRowPresentation) error {
	next := make([]string, len(rows))
	presentations := make(map[string]HostRowPresentation, len(rows))
	for index, row := range rows {
		next[index] = row.ID
		presentations[row.ID] = row
	}
	for _, operation := range ReconcileRows(*order, next) {
		switch operation.Kind {
		case ReconcileRemove:
			if widget := widgets[operation.ID]; widget != nil {
				widget.Dispose()
				delete(widgets, operation.ID)
			}
		case ReconcileInsert:
			widget, err := NewHostRowWidget(container, v.env, presentations[operation.ID], v.selectPane)
			if err != nil {
				return fmt.Errorf("create host row %q: %w", operation.ID, err)
			}
			widgets[operation.ID] = widget
			children := container.Children()
			if at := children.Index(widget); at != operation.To {
				_ = children.Remove(widget)
				if err := children.Insert(operation.To, widget); err != nil {
					return err
				}
			}
		case ReconcileMove:
			widget := widgets[operation.ID]
			children := container.Children()
			_ = children.Remove(widget)
			if err := children.Insert(operation.To, widget); err != nil {
				return err
			}
			widget.SetPresentation(presentations[operation.ID])
		case ReconcileKeep:
			widgets[operation.ID].SetPresentation(presentations[operation.ID])
		}
	}
	*order = next
	return nil
}

func (v *SidebarView) Dispose() {
	if v.unsubscribe != nil {
		v.unsubscribe()
		v.unsubscribe = nil
	}
	v.Composite.Dispose()
}
