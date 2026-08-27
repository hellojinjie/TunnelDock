package ui

import (
	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

// Window is the Windows-native TunnelDock application shell. Runtime work
// remains outside this type; updates arriving from background operations use
// RefreshHosts, which marshals the control update to Walk's UI thread.
type Window struct {
	*walk.MainWindow

	model *app.Model

	searchBox        *walk.LineEdit
	currentHostList  *walk.ListBox
	missingHostList  *walk.ListBox
	detailTitle      *walk.Label
	detailConnection *walk.Label

	currentHosts []model.SSHHost
	missingHosts []model.SSHHost
}

func NewMainWindow(model *app.Model) (*Window, error) {
	window := &Window{model: model}

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
							Label{Text: "Select a host to view and manage its saved tunnels."},
							Label{Text: "Quick Forward"},
							Label{Text: "Create a temporary local forward for the selected host."},
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
	return window, nil
}

// RefreshHosts safely renders the latest application model snapshot after a
// background configuration refresh has completed.
func (w *Window) RefreshHosts() {
	walk.App().Synchronize(func() {
		_ = w.refreshHosts()
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
}
