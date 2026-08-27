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

	searchBox          *walk.LineEdit
	currentHostList    *walk.ListBox
	missingHostList    *walk.ListBox
	detailTitle        *walk.Label
	detailConnection   *walk.Label
	remotePort         *walk.LineEdit
	localPort          *walk.LineEdit
	remoteHost         *walk.LineEdit
	localAddress       *walk.LineEdit
	protocol           *walk.ComboBox
	advanced           *walk.Composite
	connectButton      *walk.PushButton
	validation         *walk.Label
	savedTunnelList    *walk.ListBox
	temporaryList      *walk.ListBox
	connectSavedButton *walk.PushButton
	disconnectButton   *walk.PushButton
	saveButton         *walk.PushButton
	browserButton      *walk.PushButton
	deleteButton       *walk.PushButton
	renameButton       *walk.PushButton
	editButton         *walk.PushButton
	logButton          *walk.PushButton
	settingsButton     *walk.PushButton
	settingsAction     func()

	currentHosts      []model.SSHHost
	missingHosts      []model.SSHHost
	selectedHost      *model.SSHHost
	syncingForm       bool
	selectedTunnelID  string
	selectedTemporary bool
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
							LineEdit{
								AssignTo:      &window.searchBox,
								CueBanner:     "Search SSH hosts",
								OnTextChanged: window.onSearchChanged,
							},
							Label{Text: "SSH Hosts"},
							ListBox{
								AssignTo:              &window.currentHostList,
								StretchFactor:         1,
								OnCurrentIndexChanged: window.onCurrentHostSelected,
							},
							Label{Text: "Missing Hosts"},
							ListBox{
								AssignTo:              &window.missingHostList,
								MinSize:               Size{Width: 0, Height: 90},
								OnCurrentIndexChanged: window.onMissingHostSelected,
							},
						},
					},
					Composite{
						Layout: VBox{Margins: Margins{Left: 20, Top: 18, Right: 20, Bottom: 20}, Spacing: 10},
						Children: []Widget{
							Label{Text: "Host"},
							Label{AssignTo: &window.detailTitle, Text: "Select an SSH Host"},
							Label{Text: "Effective connection"},
							Label{AssignTo: &window.detailConnection, Text: "Choose a host from the sidebar."},
							Label{Text: "Saved Tunnels"},
							ListBox{AssignTo: &window.savedTunnelList, MinSize: Size{Width: 0, Height: 90}, OnCurrentIndexChanged: window.onSavedTunnelSelected},
							Label{Text: "Temporary Tunnels"},
							ListBox{AssignTo: &window.temporaryList, MinSize: Size{Width: 0, Height: 70}, OnCurrentIndexChanged: window.onTemporaryTunnelSelected},
							Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
								PushButton{AssignTo: &window.connectSavedButton, Text: "Connect", OnClicked: window.onConnectSaved},
								PushButton{AssignTo: &window.disconnectButton, Text: "Disconnect", OnClicked: window.onDisconnect},
								PushButton{AssignTo: &window.saveButton, Text: "Save", OnClicked: window.onSaveTemporary},
								PushButton{AssignTo: &window.browserButton, Text: "Open in Browser", OnClicked: window.onOpenBrowser},
								PushButton{AssignTo: &window.deleteButton, Text: "Delete", OnClicked: window.onDelete},
								PushButton{AssignTo: &window.renameButton, Text: "Rename", OnClicked: window.onRename},
								PushButton{AssignTo: &window.editButton, Text: "Edit", OnClicked: window.onEdit},
								PushButton{AssignTo: &window.logButton, Text: "View Log", OnClicked: window.onViewLog},
								PushButton{AssignTo: &window.settingsButton, Text: "Settings", OnClicked: window.onSettings},
							}},
							Label{Text: "Quick Forward"},
							Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
								LineEdit{AssignTo: &window.remotePort, CueBanner: "Remote Port", StretchFactor: 1, OnTextChanged: window.onRemotePortChanged},
								PushButton{AssignTo: &window.connectButton, Text: "Connect", OnClicked: window.onConnect},
							}},
							PushButton{Text: "Advanced", OnClicked: window.toggleAdvanced},
							Composite{AssignTo: &window.advanced, Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
								Label{Text: "Local Port"}, LineEdit{AssignTo: &window.localPort, OnTextChanged: window.onLocalPortChanged},
								Label{Text: "Remote Host"}, LineEdit{AssignTo: &window.remoteHost, Text: window.quick.RemoteHost, OnTextChanged: window.onRemoteHostChanged},
								Label{Text: "Local Address"}, LineEdit{AssignTo: &window.localAddress, Text: window.quick.LocalAddress, OnTextChanged: window.onLocalAddressChanged},
								Label{Text: "Browser Protocol"}, ComboBox{AssignTo: &window.protocol, Model: []string{"http", "https"}, OnCurrentIndexChanged: window.onProtocolChanged},
							}},
							Label{AssignTo: &window.validation},
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
	window.connectSavedButton.SetEnabled(false)
	window.disconnectButton.SetEnabled(false)
	window.saveButton.SetEnabled(false)
	window.browserButton.SetEnabled(false)
	window.deleteButton.SetEnabled(false)
	window.renameButton.SetEnabled(false)
	window.editButton.SetEnabled(false)
	window.logButton.SetEnabled(false)
	window.settingsButton.SetEnabled(false)
	return window, nil
}

// SetSettingsAction makes the tray setting reachable from the main window even
// after the user has hidden the notification icon.
func (w *Window) SetSettingsAction(action func()) {
	w.settingsAction = action
	w.settingsButton.SetEnabled(action != nil)
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
	if index < 0 || index >= len(w.currentHosts) {
		return
	}
	w.selectHost(&w.currentHosts[index])
}

func (w *Window) onMissingHostSelected() {
	index := w.missingHostList.CurrentIndex()
	if index < 0 || index >= len(w.missingHosts) {
		return
	}
	w.selectHost(&w.missingHosts[index])
}

func (w *Window) refreshHosts() error {
	w.currentHosts, w.missingHosts = PartitionHosts(w.model.FilteredHosts())
	if err := w.currentHostList.SetModel(hostAliases(w.currentHosts)); err != nil {
		return err
	}
	if err := w.missingHostList.SetModel(hostAliases(w.missingHosts)); err != nil {
		return err
	}
	w.selectHost(nil)
	return nil
}

func (w *Window) selectHost(host *model.SSHHost) {
	detail := HostDetailFor(host)
	_ = w.detailTitle.SetText(detail.Title)
	_ = w.detailConnection.SetText(detail.Connection)
	if host == nil {
		w.selectedHost = nil
		w.connectButton.SetEnabled(false)
		return
	}
	selected := *host
	w.selectedHost = &selected
	w.connectButton.SetEnabled(host.Availability == model.HostAvailable && w.manager != nil)
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
	if w.manager == nil {
		return w.savedTunnelList.SetModel([]string{})
	}
	rows := TunnelListRows(w.manager.Snapshots())
	if err := w.savedTunnelList.SetModel(rows.Saved); err != nil {
		return err
	}
	return w.temporaryList.SetModel(rows.Temporary)
}

func (w *Window) onSavedTunnelSelected() {
	index := w.savedTunnelList.CurrentIndex()
	saved := savedSnapshots(w.manager)
	if index < 0 || index >= len(saved) {
		return
	}
	w.selectedTunnelID, w.selectedTemporary = saved[index].ID, false
	w.connectSavedButton.SetEnabled(saved[index].State == model.StateDisconnected)
	w.disconnectButton.SetEnabled(saved[index].State != model.StateDisconnected)
	w.saveButton.SetEnabled(false)
	w.browserButton.SetEnabled(true)
	w.deleteButton.SetEnabled(saved[index].State == model.StateDisconnected)
	w.renameButton.SetEnabled(true)
	w.editButton.SetEnabled(saved[index].State == model.StateDisconnected)
	w.logButton.SetEnabled(true)
}

func (w *Window) onTemporaryTunnelSelected() {
	index := w.temporaryList.CurrentIndex()
	temporary := temporarySnapshots(w.manager)
	if index < 0 || index >= len(temporary) {
		return
	}
	w.selectedTunnelID, w.selectedTemporary = temporary[index].ID, true
	w.connectSavedButton.SetEnabled(false)
	w.disconnectButton.SetEnabled(temporary[index].State != model.StateDisconnected)
	w.saveButton.SetEnabled(true)
	w.browserButton.SetEnabled(true)
	w.deleteButton.SetEnabled(false)
	w.renameButton.SetEnabled(false)
	w.editButton.SetEnabled(false)
	w.logButton.SetEnabled(true)
}

func (w *Window) onDisconnect() {
	if w.manager == nil || w.selectedTunnelID == "" {
		return
	}
	_ = w.manager.Disconnect(w.selectedTunnelID)
	_ = w.refreshTunnels()
}

func (w *Window) onConnectSaved() {
	if w.manager == nil || w.selectedTemporary || w.selectedTunnelID == "" {
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
	w.deleteButton.SetEnabled(false)
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

func (w *Window) onSettings() {
	if w.settingsAction != nil {
		w.settingsAction()
	}
}
