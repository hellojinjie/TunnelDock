package ui

import (
	"context"
	"errors"
	"fmt"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
	"github.com/tailscale/win"
)

// Window is the Windows-native TunnelDock shell. Its collections are custom
// row widgets; the model and manager remain the sources of truth.
type Window struct {
	*walk.MainWindow
	model   *app.Model
	quick   *app.QuickForward
	manager *tunnel.Manager
	env     *UIEnvironment
	ownsEnv bool

	sidebar          *SidebarView
	detailTitle      *walk.Label
	detailConnection *walk.Label
	detailStatus     *walk.Label
	tunnelsTitle     *walk.Label
	tunnelCard       *Card
	tunnelList       *TunnelListView
	quickView        *QuickForwardView
	settingsButton   *IconButton
	statusLabel      *walk.Label

	currentHosts       []model.SSHHost
	visibleTunnels     []model.TunnelRuntime
	allTunnelsSelected bool
	selectedHost       *model.SSHHost
	settingsAction     func()
	addHostAction      func()
	editConfigAction   func()
	refreshAction      func()
	unsubscribe        func()
	disposed           bool
}

func NewMainWindow(applicationModel *app.Model) (*Window, error) {
	return NewMainWindowWithConnector(applicationModel, nil)
}

func NewMainWindowWithConnector(applicationModel *app.Model, manager *tunnel.Manager) (*Window, error) {
	env, err := NewUIEnvironment()
	if err != nil {
		return nil, err
	}
	window, err := NewMainWindowWithEnvironment(applicationModel, manager, env)
	if err != nil {
		env.Dispose()
		return nil, err
	}
	window.ownsEnv = true
	return window, nil
}

func NewMainWindowWithEnvironment(applicationModel *app.Model, manager *tunnel.Manager, env *UIEnvironment) (*Window, error) {
	if applicationModel == nil {
		applicationModel = app.NewModel()
	}
	if env == nil {
		return nil, fmt.Errorf("UI environment is required")
	}
	mainWindow, err := walk.NewMainWindow()
	if err != nil {
		return nil, err
	}
	window := &Window{MainWindow: mainWindow, model: applicationModel, quick: app.NewQuickForward(), manager: manager, env: env, allTunnelsSelected: true}
	fail := func(cause error) (*Window, error) {
		window.Dispose()
		return nil, cause
	}
	if err := window.SetTitle("TunnelDock"); err != nil {
		return fail(err)
	}
	if err := window.SetMinMaxSize(walk.Size{Width: 900, Height: 600}, walk.Size{}); err != nil {
		return fail(err)
	}
	if err := window.SetSize(walk.Size{Width: 1120, Height: 760}); err != nil {
		return fail(err)
	}
	rootLayout := walk.NewVBoxLayout()
	rootLayout.SetMargins(walk.Margins{})
	rootLayout.SetSpacing(0)
	if err := window.SetLayout(rootLayout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(window.DPI())
	if err != nil {
		return fail(err)
	}
	window.SetBackground(resources.WindowBrush)
	splitter, err := walk.NewHSplitter(window)
	if err != nil {
		return fail(err)
	}
	_ = splitter.SetHandleWidth(1)
	window.sidebar, err = NewSidebarView(splitter, env, SidebarCallbacks{
		SelectPane: window.selectPane,
		Search: func(query string) {
			window.model.SetSearchQuery(query)
			_ = window.refreshHosts()
		},
		AddHost: window.onAddHost, EditConfig: window.onEditConfig, Refresh: window.onRefreshConfig,
	})
	if err != nil {
		return fail(err)
	}
	if err := window.sidebar.SetMinMaxSize(walk.Size{Width: 210}, walk.Size{Width: 320}); err != nil {
		return fail(err)
	}
	_ = window.sidebar.SetSize(walk.Size{Width: 240})
	detailScroll, err := walk.NewScrollView(splitter)
	if err != nil {
		return fail(err)
	}
	detailScroll.SetScrollbars(false, true)
	detailLayout := walk.NewVBoxLayout()
	detailLayout.SetMargins(walk.Margins{HNear: resources.Metrics.PageMargin, VNear: resources.Metrics.PageMargin, HFar: resources.Metrics.PageMargin, VFar: resources.Metrics.PageMargin})
	detailLayout.SetSpacing(12)
	if err := detailScroll.SetLayout(detailLayout); err != nil {
		return fail(err)
	}
	header, err := walk.NewComposite(detailScroll)
	if err != nil {
		return fail(err)
	}
	headerLayout := walk.NewHBoxLayout()
	headerLayout.SetMargins(walk.Margins{})
	headerLayout.SetSpacing(8)
	if err := header.SetLayout(headerLayout); err != nil {
		return fail(err)
	}
	window.detailTitle, err = walk.NewLabel(header)
	if err != nil {
		return fail(err)
	}
	_ = window.detailTitle.SetText("All Tunnels")
	window.detailTitle.SetFont(resources.TitleFont)
	headerSpacer, err := walk.NewHSpacer(header)
	if err != nil {
		return fail(err)
	}
	_ = headerLayout.SetStretchFactor(headerSpacer, 1)
	window.settingsButton, err = NewIconButton(header, env, IconSettings, "Settings", nil)
	if err != nil {
		return fail(err)
	}
	window.detailConnection, err = walk.NewLabel(detailScroll)
	if err != nil {
		return fail(err)
	}
	window.detailStatus, err = walk.NewLabel(detailScroll)
	if err != nil {
		return fail(err)
	}
	window.detailStatus.SetFont(resources.CaptionFont)
	window.tunnelCard, err = NewCard(detailScroll, env)
	if err != nil {
		return fail(err)
	}
	tunnelCardLayout := walk.NewVBoxLayout()
	tunnelCardLayout.SetMargins(walk.Margins{HNear: 16, VNear: 14, HFar: 16, VFar: 14})
	tunnelCardLayout.SetSpacing(10)
	if err := window.tunnelCard.Content.SetLayout(tunnelCardLayout); err != nil {
		return fail(err)
	}
	window.tunnelsTitle, err = walk.NewLabel(window.tunnelCard.Content)
	if err != nil {
		return fail(err)
	}
	_ = window.tunnelsTitle.SetText("All Tunnels")
	window.tunnelsTitle.SetFont(resources.MediumFont)
	window.tunnelList, err = NewTunnelListView(window.tunnelCard.Content, env, TunnelRowCallbacks{
		Primary: window.onTunnelPrimary, OpenBrowser: window.onOpenBrowser,
		ViewLog: window.onViewLog, Save: window.onSaveTemporary, Rename: window.onRename,
		Edit: window.onEdit, Delete: window.onDelete,
	})
	if err != nil {
		return fail(err)
	}
	if err := window.tunnelList.SetMinMaxSize(walk.Size{Height: 180}, walk.Size{}); err != nil {
		return fail(err)
	}
	_ = tunnelCardLayout.SetStretchFactor(window.tunnelList, 1)
	window.quickView, err = NewQuickForwardView(detailScroll, env, window.quick, window.onConnect)
	if err != nil {
		return fail(err)
	}
	window.quickView.SetVisible(false)
	window.statusLabel, err = walk.NewLabel(detailScroll)
	if err != nil {
		return fail(err)
	}
	window.statusLabel.SetFont(resources.CaptionFont)
	window.statusLabel.SetTextColor(resources.Palette.Failure)
	window.statusLabel.SetVisible(false)
	if applicationIcon, iconErr := walk.NewIconFromResource("APP"); iconErr == nil {
		_ = window.SetIcon(applicationIcon)
	}
	if err := env.ApplyNativeFont(window, window.DPI()); err != nil {
		return fail(err)
	}
	window.detailTitle.SetFont(resources.TitleFont)
	window.tunnelsTitle.SetFont(resources.MediumFont)
	window.unsubscribe = env.Subscribe(func(Appearance) {
		if refreshed, resourceErr := env.Resources(window.DPI()); resourceErr == nil {
			window.SetBackground(refreshed.WindowBrush)
			window.statusLabel.SetTextColor(refreshed.Palette.Failure)
			window.Invalidate()
		}
	})
	if err := window.refreshHosts(); err != nil {
		return fail(err)
	}
	if err := window.refreshTunnels(); err != nil {
		return fail(err)
	}
	return window, nil
}

func (w *Window) Environment() *UIEnvironment { return w.env }

func (w *Window) SetSettingsAction(action func()) {
	w.settingsAction = action
	w.settingsButton.SetOnClick(action)
}

func (w *Window) SetSidebarActions(add, editConfig, refresh func()) {
	w.addHostAction, w.editConfigAction, w.refreshAction = add, editConfig, refresh
}

func (w *Window) RefreshHosts()   { walk.App().Synchronize(func() { _ = w.refreshHosts() }) }
func (w *Window) RefreshTunnels() { walk.App().Synchronize(func() { _ = w.refreshTunnels() }) }

func (w *Window) refreshHosts() error {
	w.currentHosts = w.model.FilteredHosts()
	allHosts := w.model.Hosts()
	var runtimes []model.TunnelRuntime
	if w.manager != nil {
		runtimes = w.manager.Snapshots()
	}
	active := make(map[string]bool)
	for _, runtime := range runtimes {
		if runtime.State == model.StateConnecting || runtime.State == model.StateConnected || runtime.State == model.StateReconnecting {
			active[runtime.Definition.HostAlias] = true
		}
	}
	if err := w.sidebar.SetRows(PresentHostRows(w.currentHosts, active), PresentMissingHostRows(allHosts, runtimes, w.model.SearchQuery())); err != nil {
		return err
	}
	selected := w.sidebar.Selected()
	if selected == allTunnelsPaneID {
		w.selectAllTunnels()
	} else {
		w.selectPane(selected)
	}
	return nil
}

func (w *Window) refreshTunnels() error {
	if w.manager == nil {
		w.visibleTunnels = nil
		w.tunnelList.SetEmptyText("No tunnels")
		return w.tunnelList.SetRows(nil)
	}
	snapshots := w.manager.Snapshots()
	w.visibleTunnels = snapshots
	if !w.allTunnelsSelected && w.selectedHost != nil {
		w.visibleTunnels = TunnelsForHost(snapshots, w.selectedHost.Alias)
	}
	if w.allTunnelsSelected {
		w.tunnelList.SetEmptyText("Connect a host to see its tunnels here.")
	} else {
		w.tunnelList.SetEmptyText("No tunnels for this host yet.")
	}
	return w.tunnelList.SetRows(PresentTunnelRows(w.visibleTunnels, w.model.Hosts()))
}

func (w *Window) selectPane(id string) {
	if id == allTunnelsPaneID || id == "" {
		w.selectAllTunnels()
		return
	}
	if host, found := HostForAlias(w.model.Hosts(), id); found {
		w.selectHost(&host)
		return
	}
	w.selectHost(&model.SSHHost{Alias: id, Hostname: id, Port: 22, Availability: model.HostMissing})
}

func (w *Window) selectAllTunnels() {
	w.allTunnelsSelected, w.selectedHost = true, nil
	w.sidebar.SetSelected(allTunnelsPaneID)
	_ = w.detailTitle.SetText("All Tunnels")
	_ = w.detailConnection.SetText("")
	_ = w.detailStatus.SetText("")
	_ = w.tunnelsTitle.SetText("All Tunnels")
	w.quickView.SetVisible(false)
	_ = w.refreshTunnels()
}

func (w *Window) selectHost(host *model.SSHHost) {
	w.allTunnelsSelected = false
	selected := *host
	w.selectedHost = &selected
	w.sidebar.SetSelected(host.Alias)
	detail := HostDetailFor(host)
	_ = w.detailTitle.SetText(detail.Title)
	_ = w.detailConnection.SetText(detail.Connection)
	status := hostAvailabilityText(host.Availability)
	if host.Error != "" {
		status += ": " + host.Error
	}
	_ = w.detailStatus.SetText(status)
	_ = w.tunnelsTitle.SetText("Recent Tunnels")
	available := host.Availability == model.HostAvailable
	w.quickView.SetHostAvailable(available)
	w.quickView.SetVisible(available)
	_ = w.refreshTunnels()
}

func (w *Window) onConnect() {
	if w.selectedHost == nil || w.selectedHost.Availability != model.HostAvailable || w.manager == nil {
		return
	}
	alias := w.selectedHost.Alias
	if _, err := w.quick.TunnelDefinition(alias); err != nil {
		w.quickView.SetValidation(err.Error())
		return
	}
	w.quickView.SetValidation("")
	w.quickView.SetBusy(true)
	go func() {
		_, err := w.quick.Connect(context.Background(), w.manager, alias)
		walk.App().Synchronize(func() {
			w.quickView.SetBusy(false)
			if errors.Is(err, tunnel.ErrPortUnavailable) {
				w.quick.HandlePortConflict()
				w.quickView.ApplyModelFocus()
			}
			if err != nil {
				w.showConnectionError(err, alias)
				return
			}
			w.setStatus("Tunnel connected and added to Recent Tunnels.", false)
			_ = w.refreshTunnels()
		})
	}()
}

func (w *Window) onTunnelPrimary(runtimeID string, action TunnelRowAction) {
	runtime, exists := w.runtime(runtimeID)
	if !exists {
		return
	}
	if action == TunnelRowDisconnect {
		if err := w.manager.Disconnect(runtimeID); err != nil {
			w.setStatus(err.Error(), true)
		}
		_ = w.refreshTunnels()
		return
	}
	if action != TunnelRowConnect || runtime.Temporary {
		return
	}
	w.tunnelList.SetBusy(runtimeID, true)
	go func() {
		err := w.manager.ConnectSaved(context.Background(), runtimeID)
		walk.App().Synchronize(func() {
			w.tunnelList.SetBusy(runtimeID, false)
			if err != nil {
				w.showConnectionError(err, runtime.Definition.HostAlias)
			} else {
				_ = w.refreshTunnels()
			}
		})
	}()
}

func (w *Window) onOpenBrowser(runtimeID string) {
	if runtime, ok := w.runtime(runtimeID); ok {
		if err := OpenBrowser(TunnelBrowserURL(runtime.Definition)); err != nil {
			w.setStatus(err.Error(), true)
		}
	}
}

func (w *Window) onSaveTemporary(runtimeID string) {
	if w.manager == nil {
		return
	}
	if _, err := w.manager.SaveTemporary(runtimeID); err != nil {
		w.setStatus(err.Error(), true)
		return
	}
	_ = w.refreshTunnels()
}

func (w *Window) onRename(runtimeID string) {
	runtime, exists := w.runtime(runtimeID)
	if !exists || runtime.Temporary {
		return
	}
	initial := runtime.DisplayName()
	if runtime.Definition.Name != nil {
		initial = *runtime.Definition.Name
	}
	name, accepted, err := promptTunnelRename(w, w.env, initial)
	if err != nil {
		w.setStatus(err.Error(), true)
	} else if accepted {
		if err := w.manager.Rename(runtimeID, name); err != nil {
			w.setStatus(err.Error(), true)
		} else {
			_ = w.refreshTunnels()
		}
	}
}

func (w *Window) onEdit(runtimeID string) {
	runtime, exists := w.runtime(runtimeID)
	if !exists || runtime.Temporary {
		return
	}
	definition, accepted, err := promptTunnelEdit(w, w.env, runtime.Definition)
	if err != nil {
		w.setStatus(err.Error(), true)
	} else if accepted {
		if err := w.manager.UpdateSavedDefinition(runtimeID, definition); err != nil {
			w.setStatus(err.Error(), true)
		} else {
			_ = w.refreshTunnels()
		}
	}
}

func (w *Window) onDelete(runtimeID string) {
	runtime, exists := w.runtime(runtimeID)
	if !exists || runtime.Temporary || runtime.State != model.StateDisconnected {
		return
	}
	if walk.MsgBox(w, "Delete Tunnel", "Delete "+runtime.DisplayName()+"?", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != int(win.IDYES) {
		return
	}
	if err := w.manager.Delete(runtimeID); err != nil {
		w.setStatus(err.Error(), true)
	} else {
		_ = w.refreshTunnels()
	}
}

func (w *Window) onViewLog(runtimeID string) {
	if w.manager != nil {
		if err := ShowTunnelLog(w, w.manager, runtimeID); err != nil {
			w.setStatus(err.Error(), true)
		}
	}
}

func (w *Window) runtime(runtimeID string) (model.TunnelRuntime, bool) {
	if w.manager == nil {
		return model.TunnelRuntime{}, false
	}
	return w.manager.Snapshot(runtimeID)
}

func (w *Window) showConnectionError(err error, hostAlias string) {
	w.setStatus(PresentConnectionError(err).Summary, true)
	ShowConnectionError(w, err, hostAlias)
}

func (w *Window) setStatus(message string, isError bool) {
	_ = w.statusLabel.SetText(message)
	w.statusLabel.SetVisible(message != "")
	if resources, err := w.env.Resources(w.DPI()); err == nil {
		color := resources.Palette.Success
		if isError {
			color = resources.Palette.Failure
		}
		w.statusLabel.SetTextColor(color)
	}
}

func (w *Window) onAddHost() {
	if w.addHostAction != nil {
		w.addHostAction()
	}
}
func (w *Window) onEditConfig() {
	if w.editConfigAction != nil {
		w.editConfigAction()
	}
}
func (w *Window) onRefreshConfig() {
	if w.refreshAction != nil {
		w.refreshAction()
	}
}
func (w *Window) onSettings() {
	if w.settingsAction != nil {
		w.settingsAction()
	}
}

func (w *Window) Dispose() {
	if w == nil || w.disposed {
		return
	}
	w.disposed = true
	if w.unsubscribe != nil {
		w.unsubscribe()
		w.unsubscribe = nil
	}
	if w.sidebar != nil {
		w.sidebar.Dispose()
		w.sidebar = nil
	}
	if w.tunnelList != nil {
		w.tunnelList.Dispose()
		w.tunnelList = nil
	}
	if w.tunnelCard != nil {
		w.tunnelCard.Dispose()
		w.tunnelCard = nil
	}
	if w.quickView != nil {
		w.quickView.Dispose()
		w.quickView = nil
	}
	w.MainWindow.Dispose()
	if w.ownsEnv && w.env != nil {
		w.env.Dispose()
		w.env = nil
	}
}
