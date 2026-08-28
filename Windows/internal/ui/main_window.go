package ui

import (
	"context"
	"errors"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

// Window is the Windows-native TunnelDock application shell. Runtime work
// remains outside this type; updates arriving from background operations use
// RefreshHosts, which marshals the control update to Walk's UI thread.
type Window struct {
	*walk.MainWindow

	model   *app.Model
	quick   *app.QuickForward
	manager *tunnel.Manager

	searchBox           *walk.LineEdit
	allTunnelsButton    *walk.PushButton
	currentHostList     *walk.TableView
	missingHostLabel    *walk.Label
	missingHostList     *walk.TableView
	addHostButton       *walk.ToolButton
	editConfigButton    *walk.ToolButton
	refreshConfigButton *walk.ToolButton
	detailTitle         *walk.Label
	detailConnection    *walk.Label
	detailStatus        *walk.Label
	tunnelsTitle        *walk.Label
	remotePort          *walk.LineEdit
	localPort           *walk.LineEdit
	remoteHost          *walk.LineEdit
	localAddress        *walk.LineEdit
	protocol            *walk.ComboBox
	advanced            *walk.Composite
	quickSection        *walk.Composite
	connectButton       *walk.PushButton
	validation          *walk.Label
	recentTunnelList    *walk.TableView
	noTunnelsLabel      *walk.Label
	tunnelActionButton  *walk.PushButton
	browserButton       *walk.PushButton
	moreButton          *walk.PushButton
	settingsButton      *walk.PushButton
	settingsAction      func()

	currentHosts       []model.SSHHost
	hostRows           []*HostTableRow
	missingHostRows    []*HostTableRow
	visibleTunnels     []model.TunnelRuntime
	tunnelRows         []*TunnelTableRow
	allTunnelsSelected bool
	addHostAction      func()
	editConfigAction   func()
	refreshAction      func()
	selectedHost       *model.SSHHost
	syncingForm        bool
	selectedTunnelID   string
	selectedTemporary  bool
}

func NewMainWindow(model *app.Model) (*Window, error) {
	return NewMainWindowWithConnector(model, nil)
}

func NewMainWindowWithConnector(model *app.Model, manager *tunnel.Manager) (*Window, error) {
	window := &Window{model: model, quick: app.NewQuickForward(), manager: manager}

	err := (MainWindow{
		AssignTo: &window.MainWindow,
		Title:    "TunnelDock",
		MinSize:  Size{Width: 900, Height: 600},
		Size:     Size{Width: 1120, Height: 720},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			HSplitter{
				HandleWidth: 6,
				Children: []Widget{
					Composite{
						MinSize: Size{Width: 250, Height: 0},
						MaxSize: Size{Width: 360, Height: 0},
						Layout:  VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
						Children: []Widget{
							PushButton{AssignTo: &window.allTunnelsButton, Text: "▦  All Tunnels", OnClicked: window.onAllTunnels},
							Label{Text: "SSH Hosts"},
							LineEdit{
								AssignTo:      &window.searchBox,
								CueBanner:     "Search SSH hosts",
								OnTextChanged: window.onSearchChanged,
							},
							Composite{Layout: HBox{Spacing: 4}, Children: []Widget{
								ToolButton{AssignTo: &window.addHostButton, Text: "+", ToolTipText: "Add SSH Host", OnClicked: window.onAddHost},
								ToolButton{AssignTo: &window.editConfigButton, Text: "✎", ToolTipText: "Open SSH config", OnClicked: window.onEditConfig},
								ToolButton{AssignTo: &window.refreshConfigButton, Text: "↻", ToolTipText: "Refresh SSH config", OnClicked: window.onRefreshConfig},
							}},
							TableView{
								AssignTo:            &window.currentHostList,
								HeaderHidden:        true,
								LastColumnStretched: true,
								Columns: []TableViewColumn{
									{DataMember: "Alias", Width: 170},
									{DataMember: "Status", Width: 90},
								},
								Model:                 window.hostRows,
								StretchFactor:         1,
								OnCurrentIndexChanged: window.onCurrentHostSelected,
							},
							Label{AssignTo: &window.missingHostLabel, Text: "Missing Hosts"},
							TableView{
								AssignTo:            &window.missingHostList,
								HeaderHidden:        true,
								LastColumnStretched: true,
								MinSize:             Size{Width: 0, Height: 70},
								Columns: []TableViewColumn{
									{DataMember: "Alias", Width: 110},
									{DataMember: "Status", Width: 100},
								},
								Model:                 window.missingHostRows,
								OnCurrentIndexChanged: window.onMissingHostSelected,
							},
						},
					},
					Composite{
						Layout: VBox{Margins: Margins{Left: 20, Top: 18, Right: 20, Bottom: 20}, Spacing: 10},
						Children: []Widget{
							Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
								Label{AssignTo: &window.detailTitle, Text: "All Tunnels", Font: Font{PointSize: 18, Bold: true}},
								HSpacer{},
								PushButton{AssignTo: &window.settingsButton, Text: "Settings", OnClicked: window.onSettings},
							}},
							Label{AssignTo: &window.detailConnection, Text: "Choose a host from the sidebar."},
							Label{AssignTo: &window.detailStatus},
							Label{AssignTo: &window.tunnelsTitle, Text: "All Tunnels", Font: Font{Bold: true}},
							Label{AssignTo: &window.noTunnelsLabel, Text: "No tunnels for this host yet."},
							TableView{
								AssignTo:            &window.recentTunnelList,
								HeaderHidden:        true,
								LastColumnStretched: true,
								MinSize:             Size{Width: 0, Height: 150},
								Columns: []TableViewColumn{
									{DataMember: "Name", Width: 120},
									{DataMember: "Forward", Width: 260},
									{DataMember: "Status", Width: 100},
								},
								Model:                 window.tunnelRows,
								OnCurrentIndexChanged: window.onRecentTunnelSelected,
							},
							Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
								PushButton{AssignTo: &window.tunnelActionButton, Text: "Connect", OnClicked: window.onTunnelAction},
								PushButton{AssignTo: &window.browserButton, Text: "Open in Browser", OnClicked: window.onOpenBrowser},
								PushButton{AssignTo: &window.moreButton, Text: "More…", OnClicked: window.onMore},
							}},
							Composite{AssignTo: &window.quickSection, Layout: VBox{Spacing: 10}, Children: []Widget{
								Label{Text: "Quick Forward"},
								Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
									LineEdit{AssignTo: &window.remotePort, CueBanner: "Remote Port", StretchFactor: 1, OnTextChanged: window.onRemotePortChanged},
									PushButton{AssignTo: &window.connectButton, Text: "Connect", OnClicked: window.onConnect},
								}},
								Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
									PushButton{Text: "Advanced", OnClicked: window.toggleAdvanced},
								}},
								Composite{AssignTo: &window.advanced, Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
									Label{Text: "Local Port"}, LineEdit{AssignTo: &window.localPort, OnTextChanged: window.onLocalPortChanged},
									Label{Text: "Remote Host"}, LineEdit{AssignTo: &window.remoteHost, Text: window.quick.RemoteHost, OnTextChanged: window.onRemoteHostChanged},
									Label{Text: "Local Address"}, LineEdit{AssignTo: &window.localAddress, Text: window.quick.LocalAddress, OnTextChanged: window.onLocalAddressChanged},
									Label{Text: "Browser Protocol"}, ComboBox{AssignTo: &window.protocol, Model: []string{"http", "https"}, OnCurrentIndexChanged: window.onProtocolChanged},
								}},
								Label{AssignTo: &window.validation},
							}},
						},
					},
				},
			},
		},
	}).Create()
	if err != nil {
		return nil, err
	}

	if err := window.refreshHosts(); err != nil {
		window.Dispose()
		return nil, err
	}
	if err := window.refreshTunnels(); err != nil {
		window.Dispose()
		return nil, err
	}
	window.advanced.SetVisible(false)
	if err := window.protocol.SetCurrentIndex(0); err != nil {
		window.Dispose()
		return nil, err
	}
	window.connectButton.SetEnabled(false)
	window.tunnelActionButton.SetEnabled(false)
	window.browserButton.SetEnabled(false)
	window.moreButton.SetEnabled(false)
	window.settingsButton.SetEnabled(false)
	window.quickSection.SetVisible(false)
	return window, nil
}

// SetSettingsAction makes the tray setting reachable from the main window even
// after the user has hidden the notification icon.
func (w *Window) SetSettingsAction(action func()) {
	w.settingsAction = action
	w.settingsButton.SetEnabled(action != nil)
}

// SetSidebarActions wires the same compact add/edit/refresh controls exposed
// by the macOS sidebar without coupling the native window to file I/O.
func (w *Window) SetSidebarActions(add, editConfig, refresh func()) {
	w.addHostAction = add
	w.editConfigAction = editConfig
	w.refreshAction = refresh
	w.addHostButton.SetEnabled(add != nil)
	w.editConfigButton.SetEnabled(editConfig != nil)
	w.refreshConfigButton.SetEnabled(refresh != nil)
}

// RefreshHosts safely renders the latest application model snapshot after a
// background configuration refresh has completed.
func (w *Window) RefreshHosts() {
	walk.App().Synchronize(func() {
		_ = w.refreshHosts()
	})
}

// RefreshTunnels safely re-renders tunnel state after a background operation.
func (w *Window) RefreshTunnels() {
	walk.App().Synchronize(func() {
		_ = w.refreshTunnels()
	})
}

func (w *Window) onSearchChanged() {
	w.model.SetSearchQuery(w.searchBox.Text())
	_ = w.refreshHosts()
}

func (w *Window) onCurrentHostSelected() {
	index := w.currentHostList.CurrentIndex()
	if index < 0 || index >= len(w.hostRows) {
		return
	}
	host, found := HostForAlias(w.currentHosts, w.hostRows[index].Alias)
	if !found {
		return
	}
	w.selectHost(&host)
}

func (w *Window) onAllTunnels() {
	w.selectAllTunnels()
}

func (w *Window) onMissingHostSelected() {
	index := w.missingHostList.CurrentIndex()
	if index < 0 || index >= len(w.missingHostRows) {
		return
	}
	row := w.missingHostRows[index]
	w.selectHost(&model.SSHHost{Alias: row.Alias, Hostname: row.Alias, Port: 22, Availability: model.HostMissing})
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

func (w *Window) refreshHosts() error {
	w.currentHosts = w.model.FilteredHosts()
	w.hostRows = HostTableRows(w.currentHosts)
	if err := w.currentHostList.SetModel(w.hostRows); err != nil {
		return err
	}
	allHosts := w.model.Hosts()
	var runtimes []model.TunnelRuntime
	if w.manager != nil {
		runtimes = w.manager.Snapshots()
	}
	w.missingHostRows = MissingHostTableRows(allHosts, runtimes, w.model.SearchQuery())
	if err := w.missingHostList.SetModel(w.missingHostRows); err != nil {
		return err
	}
	w.missingHostLabel.SetVisible(len(w.missingHostRows) > 0)
	w.missingHostList.SetVisible(len(w.missingHostRows) > 0)
	if w.allTunnelsSelected || w.selectedHost == nil {
		w.selectAllTunnels()
		return nil
	}
	for index := range w.currentHosts {
		if w.currentHosts[index].Alias == w.selectedHost.Alias {
			w.selectHost(&w.currentHosts[index])
			return nil
		}
	}
	w.selectAllTunnels()
	return nil
}

func (w *Window) selectAllTunnels() {
	w.allTunnelsSelected = true
	w.selectedHost = nil
	_ = w.detailTitle.SetText("All Tunnels")
	_ = w.detailConnection.SetText("")
	_ = w.detailStatus.SetText("")
	_ = w.tunnelsTitle.SetText("All Tunnels")
	w.quickSection.SetVisible(false)
	w.refreshTunnels()
}

func (w *Window) selectHost(host *model.SSHHost) {
	w.allTunnelsSelected = false
	detail := HostDetailFor(host)
	_ = w.detailTitle.SetText(detail.Title)
	_ = w.detailConnection.SetText(detail.Connection)
	_ = w.tunnelsTitle.SetText("Recent Tunnels")
	if host == nil {
		w.selectedHost = nil
		_ = w.detailStatus.SetText("")
		w.connectButton.SetEnabled(false)
		w.quickSection.SetVisible(false)
		_ = w.refreshTunnels()
		return
	}
	selected := *host
	w.selectedHost = &selected
	status := hostAvailabilityText(host.Availability)
	if host.Error != "" {
		status += ": " + host.Error
	}
	_ = w.detailStatus.SetText(status)
	w.quickSection.SetVisible(host.Availability == model.HostAvailable)
	w.connectButton.SetEnabled(host.Availability == model.HostAvailable && w.manager != nil)
	_ = w.refreshTunnels()
}

func (w *Window) onRemotePortChanged() { w.quick.SetRemotePort(w.remotePort.Text()); w.syncLocalPort() }
func (w *Window) onLocalPortChanged() {
	if !w.syncingForm {
		w.quick.SetLocalPort(w.localPort.Text())
	}
}
func (w *Window) onRemoteHostChanged()   { w.quick.RemoteHost = w.remoteHost.Text() }
func (w *Window) onLocalAddressChanged() { w.quick.LocalAddress = w.localAddress.Text() }
func (w *Window) onProtocolChanged() {
	if w.protocol.CurrentIndex() == 1 {
		w.quick.WebProtocol = model.TunnelProtocolHTTPS
		return
	}
	w.quick.WebProtocol = model.TunnelProtocolHTTP
}

func (w *Window) syncLocalPort() {
	if !w.quick.LocalPortFollowsRemote() {
		return
	}
	w.syncingForm = true
	_ = w.localPort.SetText(w.quick.LocalPort)
	w.syncingForm = false
}

func (w *Window) toggleAdvanced() { w.advanced.SetVisible(!w.advanced.Visible()) }

func (w *Window) onConnect() {
	if w.selectedHost == nil || w.selectedHost.Availability != model.HostAvailable || w.manager == nil {
		return
	}
	if _, err := w.quick.TunnelDefinition(w.selectedHost.Alias); err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	_ = w.validation.SetText("Connecting...")
	alias := w.selectedHost.Alias
	go func() {
		_, err := w.quick.Connect(context.Background(), w.manager, alias)
		walk.App().Synchronize(func() {
			if errors.Is(err, tunnel.ErrPortUnavailable) {
				w.quick.HandlePortConflict()
				w.advanced.SetVisible(true)
				_ = w.localPort.SetFocus()
			}
			if err != nil {
				_ = w.validation.SetText(err.Error())
			} else {
				_ = w.validation.SetText("Temporary tunnel is connecting.")
				_ = w.refreshTunnels()
			}
		})
	}()
}

func (w *Window) refreshTunnels() error {
	w.selectedTunnelID = ""
	w.selectedTemporary = false
	_ = w.tunnelActionButton.SetText("Connect")
	w.tunnelActionButton.SetEnabled(false)
	w.browserButton.SetEnabled(false)
	w.moreButton.SetEnabled(false)
	if w.manager == nil {
		w.visibleTunnels = nil
		w.tunnelRows = nil
		w.noTunnelsLabel.SetVisible(true)
		return w.recentTunnelList.SetModel(w.tunnelRows)
	}
	alias := ""
	if w.selectedHost != nil {
		alias = w.selectedHost.Alias
	}
	w.visibleTunnels = TunnelsForHost(w.manager.Snapshots(), alias)
	if w.allTunnelsSelected {
		w.visibleTunnels = w.manager.Snapshots()
	}
	w.tunnelRows = TunnelTableRows(w.visibleTunnels)
	if w.allTunnelsSelected {
		_ = w.noTunnelsLabel.SetText("Connect a host to see its tunnel here.")
	} else {
		_ = w.noTunnelsLabel.SetText("No tunnels for this host yet.")
	}
	w.noTunnelsLabel.SetVisible(alias != "" && len(w.visibleTunnels) == 0)
	if w.allTunnelsSelected {
		w.noTunnelsLabel.SetVisible(len(w.visibleTunnels) == 0)
	}
	return w.recentTunnelList.SetModel(w.tunnelRows)
}

func (w *Window) onRecentTunnelSelected() {
	index := w.recentTunnelList.CurrentIndex()
	if index < 0 || index >= len(w.tunnelRows) {
		return
	}
	runtime, found := TunnelForRuntimeID(w.visibleTunnels, w.tunnelRows[index].RuntimeID)
	if !found {
		return
	}
	w.selectedTunnelID, w.selectedTemporary = runtime.ID, runtime.Temporary
	if runtime.State == model.StateDisconnected {
		_ = w.tunnelActionButton.SetText("Connect")
		w.tunnelActionButton.SetEnabled(!runtime.Temporary)
	} else {
		_ = w.tunnelActionButton.SetText("Disconnect")
		w.tunnelActionButton.SetEnabled(true)
	}
	w.browserButton.SetEnabled(runtime.State == model.StateConnected)
	w.moreButton.SetEnabled(true)
}

func (w *Window) onTunnelAction() {
	if w.manager == nil || w.selectedTemporary || w.selectedTunnelID == "" {
		if w.manager != nil && w.selectedTemporary && w.selectedTunnelID != "" {
			_ = w.manager.Disconnect(w.selectedTunnelID)
			_ = w.refreshTunnels()
		}
		return
	}
	runtime, exists := w.manager.Snapshot(w.selectedTunnelID)
	if !exists {
		return
	}
	if runtime.State != model.StateDisconnected {
		_ = w.manager.Disconnect(w.selectedTunnelID)
		_ = w.refreshTunnels()
		return
	}
	id := w.selectedTunnelID
	go func() {
		err := w.manager.ConnectSaved(context.Background(), id)
		walk.App().Synchronize(func() {
			if err != nil {
				_ = w.validation.SetText(err.Error())
			} else {
				_ = w.refreshTunnels()
			}
		})
	}()
}

func (w *Window) onSaveTemporary() {
	if w.manager == nil || !w.selectedTemporary || w.selectedTunnelID == "" {
		return
	}
	if _, err := w.manager.SaveTemporary(w.selectedTunnelID); err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	_ = w.refreshTunnels()
}

func (w *Window) onOpenBrowser() {
	if w.manager == nil || w.selectedTunnelID == "" {
		return
	}
	runtime, exists := w.manager.Snapshot(w.selectedTunnelID)
	if !exists {
		return
	}
	if err := OpenBrowser(TunnelBrowserURL(runtime.Definition)); err != nil {
		_ = w.validation.SetText(err.Error())
	}
}

func (w *Window) onDelete() {
	if w.manager == nil || w.selectedTemporary || w.selectedTunnelID == "" {
		return
	}
	if walk.MsgBox(w, "Delete Tunnel", "Delete this saved tunnel?", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != int(win.IDYES) {
		return
	}
	if err := w.manager.Delete(w.selectedTunnelID); err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	w.selectedTunnelID = ""
	_ = w.refreshTunnels()
}

func (w *Window) onRename() {
	if w.manager == nil || w.selectedTemporary || w.selectedTunnelID == "" {
		return
	}
	runtime, exists := w.manager.Snapshot(w.selectedTunnelID)
	if !exists {
		return
	}
	initial := runtime.DisplayName()
	if runtime.Definition.Name != nil {
		initial = *runtime.Definition.Name
	}
	name, accepted, err := promptTunnelRename(w, initial)
	if err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	if !accepted {
		return
	}
	if err := w.manager.Rename(w.selectedTunnelID, name); err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	_ = w.refreshTunnels()
}

func (w *Window) onEdit() {
	if w.manager == nil || w.selectedTemporary || w.selectedTunnelID == "" {
		return
	}
	runtime, exists := w.manager.Snapshot(w.selectedTunnelID)
	if !exists {
		return
	}
	definition, accepted, err := promptTunnelEdit(w, runtime.Definition)
	if err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	if !accepted {
		return
	}
	if err := w.manager.UpdateSavedDefinition(w.selectedTunnelID, definition); err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	_ = w.refreshTunnels()
}

func (w *Window) onViewLog() {
	if w.manager == nil || w.selectedTunnelID == "" {
		return
	}
	if err := ShowTunnelLog(w, w.manager, w.selectedTunnelID); err != nil {
		_ = w.validation.SetText(err.Error())
	}
}

func (w *Window) onMore() {
	if w.manager == nil || w.selectedTunnelID == "" {
		return
	}
	runtime, exists := w.manager.Snapshot(w.selectedTunnelID)
	if !exists {
		return
	}
	action, err := promptTunnelMore(w, runtime)
	if err != nil {
		_ = w.validation.SetText(err.Error())
		return
	}
	switch action {
	case tunnelMoreSave:
		w.onSaveTemporary()
	case tunnelMoreRename:
		w.onRename()
	case tunnelMoreEdit:
		w.onEdit()
	case tunnelMoreDelete:
		w.onDelete()
	case tunnelMoreLog:
		w.onViewLog()
	}
}

func (w *Window) onSettings() {
	if w.settingsAction != nil {
		w.settingsAction()
	}
}
