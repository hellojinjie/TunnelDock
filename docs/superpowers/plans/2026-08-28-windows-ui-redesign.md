# TunnelDock Windows UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Rebuild the complete Windows interface with macOS-style hierarchy, custom host and tunnel rows, unified dialogs, and live light/dark system-theme support without changing TunnelDock's backend behavior.

**Architecture:** Keep the existing app.Model and tunnel.Manager as the sources of truth. Add immutable presentation models, a shared UIEnvironment for theme and resource ownership, and DPI-aware Walk CustomWidgetPixels rows hosted by ScrollView controls. Keep native inputs, menus, modality, and window frames, while replacing TableView-based content and detached row actions.

**Tech Stack:** Go 1.27, github.com/tailscale/walk, github.com/tailscale/win, golang.org/x/sys/windows and windows/registry, Win32 GDI/DWM/UxTheme, standard Go testing

**Spec:** docs/superpowers/specs/2026-08-28-windows-ui-redesign.md

## Global Constraints

- Change only the Windows implementation and its Windows documentation.
- Do not change the saved tunnel schema or runtime semantics.
- Do not change SSH discovery, process, security, or lifecycle behavior.
- Continue using github.com/tailscale/walk; do not add WebView, HTML UI, Electron, Wails, Fyne, CGO, or another widget toolkit.
- The main-window host and tunnel collections must not use TableView or ListView.
- Keep the Windows title bar and operating-system window behavior native.
- Support live light and dark Windows app-theme changes.
- Scale custom metrics and resources for per-monitor DPI.
- Keep native text fields, combo boxes, popup menus, modality, and keyboard traversal.
- Perform all widget access on the Walk UI thread.
- Dispose every GDI, Walk font, brush, pen, bitmap, icon, watcher, timer, and window owned by the new UI layer.
- Existing Windows tests, go vet ./..., the release build, and macOS source isolation must remain intact.

## File map

Create:

- Windows/internal/ui/presentation.go — immutable host, tunnel, and page presentation models.
- Windows/internal/ui/presentation_test.go — presentation and action-state tests.
- Windows/internal/ui/theme.go — semantic palettes and DPI-scaled metrics.
- Windows/internal/ui/theme_test.go — palette and scaling tests.
- Windows/internal/ui/theme_windows.go — registry-backed system appearance source and watcher.
- Windows/internal/ui/environment.go — shared theme subscriptions and disposable UI resources.
- Windows/internal/ui/paint.go — shared drawing primitives and icons.
- Windows/internal/ui/row_layout.go — pure geometry and hit testing.
- Windows/internal/ui/row_layout_test.go — geometry and hit-test tests.
- Windows/internal/ui/reconcile.go — stable-ID row reconciliation.
- Windows/internal/ui/reconcile_test.go — insertion, removal, and reorder tests.
- Windows/internal/ui/host_row.go — custom host row widget.
- Windows/internal/ui/sidebar_view.go — custom scrollable sidebar.
- Windows/internal/ui/sidebar_view_test.go — sidebar selection-state tests.
- Windows/internal/ui/tunnel_row.go — custom tunnel row widget.
- Windows/internal/ui/tunnel_list_view.go — custom scrollable tunnel list.
- Windows/internal/ui/tunnel_list_view_test.go — tunnel callback and preservation tests.
- Windows/internal/ui/quick_forward_view.go — Quick Forward card component.
- Windows/internal/ui/quick_forward_view_test.go — Quick Forward view-state tests.
- Windows/internal/ui/card.go — themed card container used by pages and dialogs.
- Windows/internal/ui/dialog_shell.go — shared dialog construction and inline validation.
- Windows/internal/ui/dialog_shell_test.go — validation and button-policy tests.
- Windows/internal/ui/confirm_dialog.go — themed destructive confirmation.
- Windows/internal/ui/ui_smoke_windows_test.go — hidden-window construction and disposal checks.

Modify:

- Windows/internal/ui/host_sidebar.go — keep host filtering helpers but return HostRowPresentation values.
- Windows/internal/ui/host_sidebar_test.go — replace table-row assertions with presentation assertions.
- Windows/internal/ui/tunnel_list.go — keep tunnel filtering and lookup helpers but remove table-specific types.
- Windows/internal/ui/tunnel_list_test.go — replace table-row assertions with presentation assertions.
- Windows/internal/ui/main_window.go — compose the new sidebar, pages, cards, and inline row actions.
- Windows/internal/ui/edit_dialog.go — migrate to DialogShell and inline validation.
- Windows/internal/ui/rename_dialog.go — migrate to DialogShell and select initial text.
- Windows/internal/ui/host_dialog.go — migrate to DialogShell and keep invalid forms open.
- Windows/internal/ui/connection_error.go — use the shared error dialog.
- Windows/internal/ui/log_viewer.go — add themed header, monospace log view, and lifecycle cleanup.
- Windows/internal/ui/tray.go — use the unified settings dialog and shared environment.
- Windows/cmd/tunneldock/main.go — create and dispose UIEnvironment and pass it to windows and dialogs.
- Windows/docs/manual-acceptance.md — add the visual, DPI, theme, and keyboard matrix.

Delete after callers migrate:

- Windows/internal/ui/tunnel_more_dialog.go — replaced by the tunnel row's native popup menu.
- Windows/internal/ui/text_scale.go — replaced by UIEnvironment-owned fonts.

---

### Task 1: Immutable presentation models

**Files:**
- Create: Windows/internal/ui/presentation.go
- Create: Windows/internal/ui/presentation_test.go
- Modify: Windows/internal/ui/host_sidebar.go
- Modify: Windows/internal/ui/host_sidebar_test.go
- Modify: Windows/internal/ui/tunnel_list.go
- Modify: Windows/internal/ui/tunnel_list_test.go

**Interfaces:**
- Consumes: model.SSHHost, model.TunnelRuntime, HostDetailFor, tunnelStateText.
- Produces:
  - func PresentHostRows(hosts []model.SSHHost, activeAliases map[string]bool) []HostRowPresentation
  - func PresentMissingHostRows(hosts []model.SSHHost, runtimes []model.TunnelRuntime, query string) []HostRowPresentation
  - func PresentTunnelRows(runtimes []model.TunnelRuntime, hosts []model.SSHHost) []TunnelRowPresentation
  - type TunnelRowAction and TunnelRowPresentation.PrimaryAction.

- [ ] **Step 1: Write failing presentation tests**

~~~go
func TestPresentTunnelRowsOwnsInlineActions(t *testing.T) {
	rows := PresentTunnelRows(
		[]model.TunnelRuntime{{
			ID: "jupyter",
			Definition: model.TunnelDefinition{
				HostAlias: "gpu", Name: stringPtr("Jupyter"),
				LocalAddress: "127.0.0.1", LocalPort: 8888,
				RemoteHost: "127.0.0.1", RemotePort: 8888,
			},
			State: model.StateConnected,
		}},
		[]model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}},
	)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ID != "jupyter" || row.Name != "Jupyter" || row.StateText != "Connected" {
		t.Fatalf("row = %#v", row)
	}
	if row.PrimaryAction != TunnelRowDisconnect || !row.ShowBrowser || !row.ShowMore {
		t.Fatalf("actions = %#v", row)
	}
}

func TestPresentFailedTunnelIncludesInlineError(t *testing.T) {
	rows := PresentTunnelRows([]model.TunnelRuntime{{
		ID: "failed", State: model.StateFailed, LastError: "Permission denied",
		Definition: model.TunnelDefinition{HostAlias: "gpu", LocalPort: 9000, RemotePort: 9000},
	}}, []model.SSHHost{{Alias: "gpu", Availability: model.HostAvailable}})
	if rows[0].ErrorText != "Permission denied" || rows[0].PrimaryAction != TunnelRowConnect {
		t.Fatalf("row = %#v", rows[0])
	}
}
~~~

- [ ] **Step 2: Run the focused tests and verify failure**

Run from Windows:

~~~powershell
go test ./internal/ui -run "TestPresent(TunnelRowsOwnsInlineActions|FailedTunnelIncludesInlineError)" -count=1
~~~

Expected: FAIL because PresentTunnelRows and the presentation types do not exist.

- [ ] **Step 3: Implement the presentation types and conversions**

~~~go
type TunnelRowAction uint8

const (
	TunnelRowNoAction TunnelRowAction = iota
	TunnelRowConnect
	TunnelRowDisconnect
	TunnelRowOpenBrowser
	TunnelRowMore
)

type HostRowPresentation struct {
	ID           string
	Title        string
	Availability model.HostAvailability
	Active       bool
	Missing      bool
}

type TunnelRowPresentation struct {
	ID            string
	HostAlias     string
	Name          string
	Forward       string
	ErrorText     string
	State         model.TunnelState
	StateText     string
	Temporary     bool
	PrimaryAction TunnelRowAction
	PrimaryText   string
	ShowBrowser   bool
	ShowMore      bool
	CanConnect    bool
}
~~~

Make PresentTunnelRows derive every action state from the runtime plus the matching host's availability. Keep TunnelsForHost and TunnelForRuntimeID as pure lookup helpers. Retain HostTableRow, TunnelTableRow, HostTableRows, TunnelTableRows, and TunnelListRows as temporary compatibility adapters because the old main window still consumes them. Task 8 removes those adapters immediately after switching the main window to custom lists.

- [ ] **Step 4: Run all UI package tests**

Run:

~~~powershell
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 5: Commit**

~~~powershell
git add Windows/internal/ui/presentation.go Windows/internal/ui/presentation_test.go Windows/internal/ui/host_sidebar.go Windows/internal/ui/host_sidebar_test.go Windows/internal/ui/tunnel_list.go Windows/internal/ui/tunnel_list_test.go
git commit -m "refactor(windows): add immutable UI presentations"
~~~

---

### Task 2: Theme tokens, DPI metrics, and system appearance

**Files:**
- Create: Windows/internal/ui/theme.go
- Create: Windows/internal/ui/theme_test.go
- Create: Windows/internal/ui/theme_windows.go
- Create: Windows/internal/ui/environment.go

**Interfaces:**
- Consumes: walk.Color, walk.Font, windows/registry.
- Produces:
  - type Appearance with AppearanceLight and AppearanceDark.
  - func PaletteFor(Appearance) Palette.
  - func MetricsForDPI(dpi int) Metrics.
  - func NewUIEnvironment() (*UIEnvironment, error).
  - func (e *UIEnvironment) Subscribe(func(Appearance)) func().
  - func (e *UIEnvironment) Resources(dpi int) (*UIResources, error).
  - func (e *UIEnvironment) Dispose().

- [ ] **Step 1: Write failing token and metric tests**

~~~go
func TestPaletteForProvidesDistinctReadableThemes(t *testing.T) {
	light := PaletteFor(AppearanceLight)
	dark := PaletteFor(AppearanceDark)
	if light.Window == dark.Window || light.PrimaryText == dark.PrimaryText {
		t.Fatalf("light and dark palettes must differ")
	}
	if contrastRatio(light.PrimaryText, light.Window) < 4.5 {
		t.Fatalf("light primary contrast is below 4.5")
	}
	if contrastRatio(dark.PrimaryText, dark.Window) < 4.5 {
		t.Fatalf("dark primary contrast is below 4.5")
	}
}

func TestMetricsForDPIScalesLogicalValues(t *testing.T) {
	at96 := MetricsForDPI(96)
	at144 := MetricsForDPI(144)
	if at96.CardRadius != 8 || at96.PageMargin != 24 {
		t.Fatalf("96 DPI metrics = %#v", at96)
	}
	if at144.CardRadius != 12 || at144.PageMargin != 36 {
		t.Fatalf("144 DPI metrics = %#v", at144)
	}
}
~~~

- [ ] **Step 2: Run the focused tests and verify failure**

Run:

~~~powershell
go test ./internal/ui -run "Test(PaletteFor|MetricsForDPI)" -count=1
~~~

Expected: FAIL because Appearance, PaletteFor, and MetricsForDPI are undefined.

- [ ] **Step 3: Implement pure palettes and scaling**

~~~go
type Appearance uint8

const (
	AppearanceLight Appearance = iota
	AppearanceDark
)

type Palette struct {
	Window, Sidebar, Surface, SurfaceHover, SurfaceSelected walk.Color
	Border, PrimaryText, SecondaryText, DisabledText        walk.Color
	Accent, Success, Connecting, Warning, Failure, Focus    walk.Color
}

type Metrics struct {
	PageMargin, SidebarPadding, CardRadius, RowRadius int
	HostRowHeight, TunnelRowHeight, TunnelErrorHeight int
	IconSize, ActionHeight, FocusWidth                int
}

func scaleMetric(value, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return (value*dpi + 48) / 96
}
~~~

Use explicit semantic values for both palettes. Implement contrastRatio only in the test file.

- [ ] **Step 4: Implement the Windows appearance source**

Read HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize\AppsUseLightTheme. Missing or unreadable values fall back to light. Watch the key with RegNotifyChangeKeyValue through advapi32, coalesce duplicate values, and stop when the environment closes.

~~~go
type appearanceSource interface {
	Current() Appearance
	Watch(context.Context) <-chan Appearance
}

type UIEnvironment struct {
	source      appearanceSource
	appearance  Appearance
	subscribers map[int]func(Appearance)
	cancel      context.CancelFunc
}
~~~

UIEnvironment marshals subscriber callbacks through walk.App().Synchronize. UIResources caches fonts, solid brushes, and pens by appearance and DPI; Dispose releases them exactly once. Add e.ApplyNativeFont(window, dpi), but leave ApplyStandardTextScale unchanged until every old dialog caller migrates in Tasks 8 through 10.

- [ ] **Step 5: Run theme tests and the full UI package**

Run:

~~~powershell
go test ./internal/ui -run "Test(PaletteFor|MetricsForDPI|UIEnvironment)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 6: Commit**

~~~powershell
git add Windows/internal/ui/theme.go Windows/internal/ui/theme_test.go Windows/internal/ui/theme_windows.go Windows/internal/ui/environment.go
git commit -m "feat(windows): add adaptive UI theme system"
~~~

---

### Task 3: Drawing primitives, icons, row geometry, and hit testing

**Files:**
- Create: Windows/internal/ui/paint.go
- Create: Windows/internal/ui/row_layout.go
- Create: Windows/internal/ui/row_layout_test.go

**Interfaces:**
- Consumes: UIResources, Palette, Metrics, walk.Canvas.
- Produces:
  - type IconKind and func DrawIcon(canvas *walk.Canvas, resources *UIResources, kind IconKind, bounds walk.Rectangle, color walk.Color) error.
  - func TunnelRowHeight(TunnelRowPresentation, Metrics) int.
  - func LayoutTunnelRow(width, height int, Metrics, TunnelRowPresentation) TunnelRowLayout.
  - func (TunnelRowLayout) HitTest(x, y int) TunnelRowAction.

- [ ] **Step 1: Write failing geometry tests**

~~~go
func TestTunnelRowHeightExpandsForError(t *testing.T) {
	metrics := MetricsForDPI(96)
	normal := TunnelRowPresentation{ID: "normal"}
	failed := TunnelRowPresentation{ID: "failed", ErrorText: "Permission denied"}
	if got := TunnelRowHeight(normal, metrics); got != metrics.TunnelRowHeight {
		t.Fatalf("normal height = %d", got)
	}
	if got := TunnelRowHeight(failed, metrics); got != metrics.TunnelErrorHeight {
		t.Fatalf("failed height = %d", got)
	}
}

func TestTunnelRowHitTestSeparatesInlineActions(t *testing.T) {
	row := TunnelRowPresentation{ShowBrowser: true, ShowMore: true, PrimaryAction: TunnelRowDisconnect}
	layout := LayoutTunnelRow(720, 68, MetricsForDPI(96), row)
	if got := layout.HitTest(layout.Browser.X+1, layout.Browser.Y+1); got != TunnelRowOpenBrowser {
		t.Fatalf("browser hit = %v", got)
	}
	if got := layout.HitTest(layout.Primary.X+1, layout.Primary.Y+1); got != TunnelRowDisconnect {
		t.Fatalf("primary hit = %v", got)
	}
	if got := layout.HitTest(layout.More.X+1, layout.More.Y+1); got != TunnelRowMore {
		t.Fatalf("more hit = %v", got)
	}
}
~~~

- [ ] **Step 2: Run the tests and verify failure**

Run:

~~~powershell
go test ./internal/ui -run "TestTunnelRow(Height|HitTest)" -count=1
~~~

Expected: FAIL because the layout API is missing.

- [ ] **Step 3: Implement deterministic geometry**

~~~go
type Rect struct{ X, Y, Width, Height int }

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}

type TunnelRowLayout struct {
	StateIcon, Text, StateLabel, Browser, Primary, More Rect
	PrimaryAction TunnelRowAction
}
~~~

Lay out actions from right to left, reserve text space, and collapse the Browser rectangle to zero when ShowBrowser is false. Use only Metrics values so tests remain DPI deterministic.

- [ ] **Step 4: Implement shared drawing primitives and icons**

Use CustomWidgetPixels-compatible functions for rounded surfaces, separators, focus rings, state circles, magnifier, plus, edit, refresh, settings, browser, ellipsis, server, warning, and tunnel-stack icons. Draw with Canvas FillRoundedRectanglePixels, DrawRoundedRectanglePixels, DrawLinePixels, FillEllipsePixels, and DrawTextPixels. Do not use Unicode characters as toolbar icons.

Draw primary and secondary row text with TextSingleLine, TextEndEllipsis, and TextNoPrefix. Store the unabridged name, address, and error text in the row presentation so HostRowWidget and TunnelRowWidget can expose tooltips whenever the measured text exceeds its rectangle.

- [ ] **Step 5: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "TestTunnelRow(Height|HitTest)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 6: Commit**

~~~powershell
git add Windows/internal/ui/paint.go Windows/internal/ui/row_layout.go Windows/internal/ui/row_layout_test.go
git commit -m "feat(windows): add DPI-aware UI drawing primitives"
~~~

---

### Task 4: Stable row reconciliation

**Files:**
- Create: Windows/internal/ui/reconcile.go
- Create: Windows/internal/ui/reconcile_test.go

**Interfaces:**
- Consumes: ordered stable IDs from presentation models.
- Produces:
  - type ReconcileKind with ReconcileKeep, ReconcileInsert, ReconcileMove, ReconcileRemove.
  - func ReconcileRows(current, next []string) []ReconcileOperation.

- [ ] **Step 1: Write failing reconciliation tests**

~~~go
func TestReconcileRowsKeepsIdentityAcrossReorder(t *testing.T) {
	ops := ReconcileRows([]string{"a", "b", "c"}, []string{"c", "a", "d"})
	want := []ReconcileOperation{
		{Kind: ReconcileRemove, ID: "b", From: 1, To: -1},
		{Kind: ReconcileMove, ID: "c", From: 1, To: 0},
		{Kind: ReconcileKeep, ID: "a", From: 1, To: 1},
		{Kind: ReconcileInsert, ID: "d", From: -1, To: 2},
	}
	if diff := cmpOperations(ops, want); diff != "" {
		t.Fatal(diff)
	}
}
~~~

Implement cmpOperations locally without adding a comparison dependency.

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestReconcileRowsKeepsIdentityAcrossReorder -count=1
~~~

Expected: FAIL because ReconcileRows is undefined.

- [ ] **Step 3: Implement reconciliation**

Use a map for identity lookup and a mutable working-order slice. Emit removals from highest index to lowest, then process the target order from left to right. The resulting operation order must be directly applicable to walk.WidgetList Remove and Insert calls.

- [ ] **Step 4: Run tests**

Run:

~~~powershell
go test ./internal/ui -run TestReconcileRows -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 5: Commit**

~~~powershell
git add Windows/internal/ui/reconcile.go Windows/internal/ui/reconcile_test.go
git commit -m "feat(windows): reconcile custom rows by stable identity"
~~~

---

### Task 5: Custom host rows and sidebar

**Files:**
- Create: Windows/internal/ui/host_row.go
- Create: Windows/internal/ui/sidebar_view.go
- Create: Windows/internal/ui/sidebar_view_test.go
- Modify: Windows/internal/ui/host_sidebar.go

**Interfaces:**
- Consumes: UIEnvironment, HostRowPresentation, ReconcileRows.
- Produces:
  - func NewSidebarView(parent walk.Container, env *UIEnvironment, callbacks SidebarCallbacks) (*SidebarView, error).
  - func (v *SidebarView) SetRows(current, missing []HostRowPresentation).
  - func (v *SidebarView) SetSelected(id string).
  - func (v *SidebarView) SearchText() string.

- [ ] **Step 1: Write failing sidebar state tests**

~~~go
func TestSidebarStatePreservesSelectionWhenAliasRemains(t *testing.T) {
	state := newSidebarState()
	state.SetSelected("gpu")
	state.Apply([]HostRowPresentation{{ID: "nas"}, {ID: "gpu"}})
	if state.Selected() != "gpu" {
		t.Fatalf("selected = %q", state.Selected())
	}
	state.Apply([]HostRowPresentation{{ID: "nas"}})
	if state.Selected() != allTunnelsPaneID {
		t.Fatalf("selected after removal = %q", state.Selected())
	}
}
~~~

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestSidebarStatePreservesSelectionWhenAliasRemains -count=1
~~~

Expected: FAIL because sidebarState is undefined.

- [ ] **Step 3: Implement HostRowWidget**

~~~go
type HostRowWidget struct {
	*walk.CustomWidget
	env          *UIEnvironment
	presentation HostRowPresentation
	selected     bool
	hovered      bool
	pressed      bool
	onActivate   func(string)
}

func NewHostRowWidget(parent walk.Container, env *UIEnvironment, row HostRowPresentation, activate func(string)) (*HostRowWidget, error)
func (w *HostRowWidget) SetPresentation(row HostRowPresentation)
func (w *HostRowWidget) SetSelected(selected bool)
~~~

Create it with walk.NewCustomWidgetPixels, PaintBuffered, InvalidatesOnResize, and WS_TABSTOP. Attach MouseMove, MouseDown, MouseUp, KeyDown, and FocusChanged events. Draw rounded selection, icon, alias, activity state, warning state, and focus ring from environment resources.

- [ ] **Step 4: Implement SidebarView**

Use native LineEdit for search and custom tool buttons painted with the shared icons. Put host rows in a vertical-only ScrollView. Apply ReconcileRows to its child WidgetList and row map. Preserve selected alias, focused alias, and vertical scroll position.

~~~go
type SidebarCallbacks struct {
	SelectPane func(string)
	Search     func(string)
	AddHost    func()
	EditConfig func()
	Refresh    func()
}
~~~

Define const allTunnelsPaneID = "__all_tunnels__" in sidebar_view.go. It is a UI-only identity and must not be passed to the SSH or tunnel layers.

- [ ] **Step 5: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "TestSidebar" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 6: Commit**

~~~powershell
git add Windows/internal/ui/host_row.go Windows/internal/ui/sidebar_view.go Windows/internal/ui/sidebar_view_test.go Windows/internal/ui/host_sidebar.go
git commit -m "feat(windows): replace host tables with custom sidebar rows"
~~~

---

### Task 6: Custom tunnel rows and inline actions

**Files:**
- Create: Windows/internal/ui/tunnel_row.go
- Create: Windows/internal/ui/tunnel_list_view.go
- Create: Windows/internal/ui/tunnel_list_view_test.go
- Modify: Windows/internal/ui/tunnel_list.go
- Modify: Windows/internal/ui/tunnel_actions.go
- Modify: Windows/internal/ui/tunnel_actions_test.go

**Interfaces:**
- Consumes: TunnelRowPresentation, TunnelRowLayout, ReconcileRows, UIEnvironment.
- Produces:
  - type TunnelRowCallbacks.
  - type TunnelMenuItem and TunnelMenuModel.
  - func MoreMenuItems(TunnelRowPresentation) TunnelMenuModel.
  - func NewTunnelListView(parent walk.Container, env *UIEnvironment, callbacks TunnelRowCallbacks) (*TunnelListView, error).
  - func (v *TunnelListView) SetRows([]TunnelRowPresentation).
  - func (v *TunnelListView) SetBusy(runtimeID string, busy bool).

- [ ] **Step 1: Write failing callback-policy tests**

~~~go
func TestTunnelRowCallbackPolicy(t *testing.T) {
	row := TunnelRowPresentation{
		ID: "tmp", Temporary: true, State: model.StateConnected,
		PrimaryAction: TunnelRowDisconnect, ShowBrowser: true, ShowMore: true,
	}
	menu := MoreMenuItems(row)
	if !menu.Enabled(TunnelMenuLog) || !menu.Enabled(TunnelMenuSave) {
		t.Fatalf("temporary menu = %#v", menu)
	}
	if menu.Contains(TunnelMenuRename) || menu.Contains(TunnelMenuDelete) {
		t.Fatalf("temporary menu exposes saved-only actions")
	}
}
~~~

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestTunnelRowCallbackPolicy -count=1
~~~

Expected: FAIL because MoreMenuItems is undefined.

- [ ] **Step 3: Implement TunnelRowWidget**

~~~go
type TunnelRowCallbacks struct {
	Primary     func(runtimeID string, action TunnelRowAction)
	OpenBrowser func(runtimeID string)
	Save        func(runtimeID string)
	Rename      func(runtimeID string)
	Edit        func(runtimeID string)
	Delete      func(runtimeID string)
	ViewLog     func(runtimeID string)
}
~~~

Paint all row content and action affordances in one CustomWidgetPixels. Use TunnelRowLayout for drawing and HitTest for mouse routing. On keyboard input, Tab cycles browser, primary, and more regions that are present; Enter or Space activates the focused region. Opening More creates a native walk.Menu populated from MoreMenuItems.

~~~go
type TunnelMenuItem uint8

const (
	TunnelMenuLog TunnelMenuItem = iota
	TunnelMenuSave
	TunnelMenuRename
	TunnelMenuEdit
	TunnelMenuDelete
)

type TunnelMenuEntry struct {
	Item    TunnelMenuItem
	Text    string
	Enabled bool
}

type TunnelMenuModel []TunnelMenuEntry

func (m TunnelMenuModel) Contains(item TunnelMenuItem) bool
func (m TunnelMenuModel) Enabled(item TunnelMenuItem) bool
~~~

- [ ] **Step 4: Implement TunnelListView reconciliation**

Host a vertical row stack in ScrollView with horizontal scrolling disabled. Reconcile by runtime ID, keep row widgets alive when IDs remain, update only changed presentations, and preserve scroll and focused runtime ID. Draw separators between rows inside the row widgets so no separate separator widgets disturb reconciliation.

- [ ] **Step 5: Keep the old main-window adapter compiling**

Keep promptTunnelMore plus the old main window's selectedTunnelID, selectedTemporary, tunnelActionButton, browserButton, and moreButton workflow unchanged for this commit. Keep TunnelBrowserURL and the manager action functions used by the new callbacks. Task 8 removes the compatibility UI after NewMainWindowWithEnvironment consumes TunnelListView.

- [ ] **Step 6: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "Test(TunnelRow|TunnelList)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 7: Commit**

~~~powershell
git add Windows/internal/ui/tunnel_row.go Windows/internal/ui/tunnel_list_view.go Windows/internal/ui/tunnel_list_view_test.go Windows/internal/ui/tunnel_list.go Windows/internal/ui/tunnel_actions.go Windows/internal/ui/tunnel_actions_test.go
git commit -m "feat(windows): add inline custom tunnel rows"
~~~

---

### Task 7: Quick Forward card

**Files:**
- Create: Windows/internal/ui/quick_forward_view.go
- Create: Windows/internal/ui/quick_forward_view_test.go
- Create: Windows/internal/ui/card.go

**Interfaces:**
- Consumes: app.QuickForward, tunnel.Manager, UIEnvironment.
- Produces:
  - type QuickForwardViewState.
  - func PresentQuickForward(*app.QuickForward, busy bool) QuickForwardViewState.
  - func NewQuickForwardView(parent walk.Container, env *UIEnvironment, connect func() error) (*QuickForwardView, error).
  - func (v *QuickForwardView) SetHost(alias string, available bool).
  - func (v *QuickForwardView) SetBusy(bool).

Define the view state once and use it in tests and widget updates:

~~~go
type QuickForwardViewState struct {
	ConnectText      string
	ConnectEnabled   bool
	AdvancedExpanded bool
	FocusField       app.FocusTarget
	Validation       string
}
~~~

- [ ] **Step 1: Write failing view-state tests**

~~~go
func TestPresentQuickForwardReflectsBusyAndConflict(t *testing.T) {
	quick := app.NewQuickForward()
	quick.SetRemotePort("8888")
	state := PresentQuickForward(quick, true)
	if state.ConnectText != "Connecting..." || state.ConnectEnabled {
		t.Fatalf("busy state = %#v", state)
	}
	quick.HandlePortConflict()
	state = PresentQuickForward(quick, false)
	if !state.AdvancedExpanded || state.FocusField != app.FocusLocalPort {
		t.Fatalf("conflict state = %#v", state)
	}
}
~~~

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestPresentQuickForwardReflectsBusyAndConflict -count=1
~~~

Expected: FAIL because PresentQuickForward is undefined.

- [ ] **Step 3: Implement Card and QuickForwardView**

Card provides a themed rounded border/background and an inner native Composite for controls. QuickForwardView owns the Remote Port, Local Port, Remote Host, Local Address, protocol, Advanced disclosure, validation label, and Connect button. It maps text changes to the existing app.QuickForward methods.

Use a custom disclosure header with a chevron icon. When collapsed, remove Advanced from layout instead of merely painting over it. When a port conflict is returned, call HandlePortConflict, expand Advanced, and focus the Local Port control.

- [ ] **Step 4: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "TestPresentQuickForward" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 5: Commit**

~~~powershell
git add Windows/internal/ui/card.go Windows/internal/ui/quick_forward_view.go Windows/internal/ui/quick_forward_view_test.go
git commit -m "feat(windows): add macOS-style quick forward card"
~~~

---

### Task 8: Rebuild the main window composition

**Files:**
- Modify: Windows/internal/ui/main_window.go
- Modify: Windows/internal/ui/host_detail.go
- Modify: Windows/internal/ui/host_detail_test.go
- Modify: Windows/internal/ui/host_sidebar.go
- Modify: Windows/internal/ui/host_sidebar_test.go
- Modify: Windows/internal/ui/tunnel_list.go
- Modify: Windows/internal/ui/tunnel_list_test.go
- Modify: Windows/internal/ui/tunnel_actions.go
- Modify: Windows/cmd/tunneldock/main.go
- Delete: Windows/internal/ui/tunnel_more_dialog.go
- Create: Windows/internal/ui/ui_smoke_windows_test.go

**Interfaces:**
- Consumes: UIEnvironment, SidebarView, TunnelListView, QuickForwardView, Card, presentation functions.
- Produces:
  - func NewMainWindowWithEnvironment(model *app.Model, manager *tunnel.Manager, env *UIEnvironment) (*Window, error).
  - func (w *Window) Environment() *UIEnvironment.
  - Existing NewMainWindow and NewMainWindowWithConnector wrappers remain available for tests and callers until Task 11 migrates main.go.

- [ ] **Step 1: Write the first hidden-window smoke test**

~~~go
func TestMainWindowBuildsWithoutTableViews(t *testing.T) {
	env, err := newTestUIEnvironment(AppearanceLight)
	if err != nil {
		t.Fatal(err)
	}
	defer env.Dispose()
	window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Dispose()
	if containsWidgetType(window, "*walk.TableView") || containsWidgetType(window, "*walk.ListBox") {
		t.Fatal("main window still contains a table or list control")
	}
}
~~~

The test helper walks container children recursively and compares reflect.TypeOf(child).String().

- [ ] **Step 2: Run the smoke test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestMainWindowBuildsWithoutTableViews -count=1
~~~

Expected: FAIL because NewMainWindowWithEnvironment and newTestUIEnvironment are undefined.

- [ ] **Step 3: Replace the declarative table shell**

Build the window imperatively so component ownership and error cleanup remain explicit. Retain the native MainWindow and HSplitter. Compose:

~~~text
MainWindow
  HSplitter
    SidebarView
    Detail ScrollView
      header and settings icon
      tunnel Card containing TunnelListView
      QuickForwardView when the selected host is available
~~~

Set the logical default and minimum sizes from the spec. Use the existing selection semantics: All Tunnels initially, host detail on alias selection, and fallback to All Tunnels when a selected alias disappears.

Set the sidebar ideal width to 240 logical pixels with a 210 minimum and 320 maximum. Give the detail page 24 logical pixels of outer margin and constrain its card column to approximately 760 logical pixels while allowing the surrounding ScrollView to expand.

- [ ] **Step 4: Wire the shared environment into the application entry**

Create UIEnvironment after walk.InitApp, defer Dispose after all owned UI windows, and construct the main window with NewMainWindowWithEnvironment. Keep tray and dialog signatures unchanged in this task; later tasks retrieve the same instance through mainWindow.Environment while migrating those call sites.

- [ ] **Step 5: Wire row callbacks to existing manager operations**

Move the current onTunnelAction, onOpenBrowser, onSaveTemporary, onRename, onEdit, onDelete, and onViewLog behavior into callbacks keyed by runtime ID. Background connect operations set only that row busy and synchronize completion back to the UI thread. Keep the current connection-error presentation.

- [ ] **Step 6: Preserve page state on refresh**

RefreshHosts updates sidebar presentations, keeps the selected alias when possible, and updates header details. RefreshTunnels updates TunnelListView presentations without clearing focus or scroll. All Tunnels continues to show manager snapshots; host pages use TunnelsForHost.

- [ ] **Step 7: Remove compatibility UI and run tests**

Delete promptTunnelMore, remove the detached selected-tunnel button bar, and remove HostTableRow, TunnelTableRow, HostTableRows, TunnelTableRows, and TunnelListRows with their table-specific assertions. Retain the pure presentation, filtering, and ID lookup helpers.

Run:

~~~powershell
go test ./internal/ui -run "Test(MainWindow|HostDetail)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 8: Commit**

~~~powershell
git add Windows/internal/ui/main_window.go Windows/internal/ui/host_detail.go Windows/internal/ui/host_detail_test.go Windows/internal/ui/host_sidebar.go Windows/internal/ui/host_sidebar_test.go Windows/internal/ui/tunnel_list.go Windows/internal/ui/tunnel_list_test.go Windows/internal/ui/tunnel_actions.go Windows/internal/ui/tunnel_more_dialog.go Windows/internal/ui/ui_smoke_windows_test.go Windows/cmd/tunneldock/main.go
git commit -m "feat(windows): rebuild main window with custom components"
~~~

---

### Task 9: Shared dialog shell and editable forms

**Files:**
- Create: Windows/internal/ui/dialog_shell.go
- Create: Windows/internal/ui/dialog_shell_test.go
- Modify: Windows/internal/ui/host_dialog.go
- Modify: Windows/internal/ui/host_dialog_test.go
- Modify: Windows/internal/ui/edit_dialog.go
- Modify: Windows/internal/ui/rename_dialog.go
- Modify: Windows/internal/ui/main_window.go
- Modify: Windows/cmd/tunneldock/main.go

**Interfaces:**
- Consumes: UIEnvironment, Card, existing SSHHostInput validation and tunnel definition validation.
- Produces:
  - func NewDialogShell(owner walk.Form, env *UIEnvironment, spec DialogSpec) (*DialogShell, error).
  - func (s *DialogShell) SetValidation(message string, field walk.Widget).
  - func ShowAddHostDialog(owner walk.Form, env *UIEnvironment, submit func(SSHHostInput) error).
  - func promptTunnelEdit(owner walk.Form, env *UIEnvironment, initial model.TunnelDefinition) (model.TunnelDefinition, bool, error).
  - func promptTunnelRename(owner walk.Form, env *UIEnvironment, initial string) (string, bool, error).

- [ ] **Step 1: Write failing validation-policy tests**

~~~go
func TestDialogValidationKeepsFormOpen(t *testing.T) {
	state := dialogValidationState{}
	state.Reject("Port must be between 1 and 65535", "port")
	if state.Accepted || state.Message == "" || state.FocusField != "port" {
		t.Fatalf("state = %#v", state)
	}
}
~~~

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestDialogValidationKeepsFormOpen -count=1
~~~

Expected: FAIL because dialogValidationState is undefined.

- [ ] **Step 3: Implement DialogShell**

~~~go
type DialogSpec struct {
	Title, Description, PrimaryText string
	Size, MinSize                   walk.Size
	Resizable                       bool
	Destructive                     bool
}

type DialogShell struct {
	*walk.Dialog
	Content    *walk.Composite
	Validation *walk.Label
	Primary    *walk.PushButton
	Cancel     *walk.PushButton
}
~~~

The shell creates title, description, card content, inline validation, right-aligned actions, default/cancel buttons, native modality, and theme subscription. Construction failures dispose the partially built dialog.

- [ ] **Step 4: Migrate Add Host and Edit Tunnel**

Do not attach the primary button directly to dialog.Accept. Its click handler parses and validates while the dialog remains open. On failure, call SetValidation and SetFocus on the first invalid control. On success, call submit or return the updated definition, then Accept.

Update main.go's Add Host call and main_window.go's Edit and Rename calls to pass mainWindow.Environment so the package remains compiling at the end of this task.

- [ ] **Step 5: Migrate Rename**

Select all initial text after showing the dialog, reject control characters through the model's existing rename path, and keep Enter/Esc behavior through default and cancel buttons.

- [ ] **Step 6: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "Test(DialogValidation|SSHHostInput)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 7: Commit**

~~~powershell
git add Windows/internal/ui/dialog_shell.go Windows/internal/ui/dialog_shell_test.go Windows/internal/ui/host_dialog.go Windows/internal/ui/host_dialog_test.go Windows/internal/ui/edit_dialog.go Windows/internal/ui/rename_dialog.go Windows/internal/ui/main_window.go Windows/cmd/tunneldock/main.go
git commit -m "feat(windows): unify editable dialogs"
~~~

---

### Task 10: Errors, confirmations, settings, and log viewer

**Files:**
- Create: Windows/internal/ui/confirm_dialog.go
- Modify: Windows/internal/ui/connection_error.go
- Modify: Windows/internal/ui/connection_error_test.go
- Modify: Windows/internal/ui/tray.go
- Modify: Windows/internal/ui/main_window.go
- Modify: Windows/internal/ui/log_viewer.go
- Modify: Windows/internal/ui/log_viewer_test.go
- Modify: Windows/internal/ui/ui_smoke_windows_test.go
- Modify: Windows/cmd/tunneldock/main.go

**Interfaces:**
- Consumes: DialogShell, UIEnvironment, app.TrayController, tunnel.Manager.
- Produces:
  - func ConfirmDeleteTunnel(owner walk.Form, env *UIEnvironment, name string) bool.
  - func ShowConnectionError(owner walk.Form, env *UIEnvironment, err error, hostAlias string).
  - func ShowTunnelLog(owner walk.Form, env *UIEnvironment, manager *tunnel.Manager, runtimeID string) error.
  - func PresentDeleteConfirmation(name string) DialogSpec.
  - func PresentSettings(showTray bool, appearance Appearance) SettingsPresentation.
  - func (t *Tray) ShowSettings() using the shared shell.

~~~go
type SettingsPresentation struct {
	ShowTrayIcon  bool
	AppearanceText string
}
~~~

- [ ] **Step 1: Add failing presentation tests**

~~~go
func TestDeleteConfirmationNamesTunnel(t *testing.T) {
	p := PresentDeleteConfirmation("Jupyter")
	if p.Title != "Delete Jupyter?" || !p.Destructive || p.PrimaryText != "Delete" {
		t.Fatalf("presentation = %#v", p)
	}
}

func TestSettingsPresentationExplainsSystemAppearance(t *testing.T) {
	p := PresentSettings(true, AppearanceDark)
	if !p.ShowTrayIcon || p.AppearanceText != "Appearance follows Windows (Dark)" {
		t.Fatalf("presentation = %#v", p)
	}
}
~~~

- [ ] **Step 2: Run focused tests and verify failure**

Run:

~~~powershell
go test ./internal/ui -run "Test(DeleteConfirmation|SettingsPresentation)" -count=1
~~~

Expected: FAIL because the presentation functions are undefined.

- [ ] **Step 3: Replace MsgBox error and deletion flows**

Use DialogShell for connection errors, including summary, suggested action, read-only technical details, Close, and conditional Open SSH Terminal. Use ConfirmDeleteTunnel for deletions and show the tunnel name. Keep running-tunnel deletion disabled before the confirmation is reachable.

Update main_window.go to pass its environment to errors, confirmations, and logs. Update main.go and NewTray so settings uses the same process-wide environment.

- [ ] **Step 4: Rebuild Settings**

Use a card row with the tray checkbox plus an appearance explanation. The dialog has a Close button rather than Save and Cancel. On each checkbox change, call TrayController.SetVisible and NotifyIcon.SetVisible immediately; if either operation fails, restore the previous checkbox value and show inline validation. Keep the existing safeguard that disabling the tray leaves a visible taskbar re-entry point.

- [ ] **Step 5: Rebuild the log viewer**

Create a resizable themed window with a header label, state label, and read-only TextEdit using a monospace font. Preserve the 250 ms live refresh interval. Avoid redundant SetText calls when log content is unchanged, keep the user's selection unless following the tail, and cancel the refresh context before disposing widgets.

- [ ] **Step 6: Extend smoke tests**

Construct and dispose Add Host, Edit, Rename, Settings, Error, Confirm, and Log windows with a fixed test appearance. Exercise theme update once before disposal. Do not start a real SSH process.

- [ ] **Step 7: Run tests**

Run:

~~~powershell
go test ./internal/ui -run "Test(DeleteConfirmation|SettingsPresentation|PresentConnectionError|LogText|UIWindows)" -count=1
go test ./internal/ui -count=1
~~~

Expected: PASS.

- [ ] **Step 8: Commit**

~~~powershell
git add Windows/internal/ui/confirm_dialog.go Windows/internal/ui/connection_error.go Windows/internal/ui/connection_error_test.go Windows/internal/ui/tray.go Windows/internal/ui/main_window.go Windows/internal/ui/log_viewer.go Windows/internal/ui/log_viewer_test.go Windows/internal/ui/ui_smoke_windows_test.go Windows/cmd/tunneldock/main.go
git commit -m "feat(windows): unify secondary windows and dialogs"
~~~

---

### Task 11: Application integration and live theme/DPI lifecycle

**Files:**
- Modify: Windows/cmd/tunneldock/main.go
- Modify: Windows/internal/ui/main_window.go
- Modify: Windows/internal/ui/tray.go
- Modify: Windows/internal/ui/environment.go
- Modify: Windows/internal/ui/ui_smoke_windows_test.go
- Delete: Windows/internal/ui/text_scale.go

**Interfaces:**
- Consumes: all components from Tasks 1 through 10.
- Produces: one process-wide UIEnvironment created in main and shared by all main, tray, dialog, and log windows.

- [ ] **Step 1: Add a failing environment-lifecycle test**

~~~go
func TestUIEnvironmentNotifiesOnceAndStopsAfterDispose(t *testing.T) {
	source := newFakeAppearanceSource(AppearanceLight)
	env := newUIEnvironment(source)
	var got []Appearance
	unsubscribe := env.Subscribe(func(value Appearance) { got = append(got, value) })
	source.Send(AppearanceDark)
	drainSynchronizedUI()
	unsubscribe()
	env.Dispose()
	source.Send(AppearanceLight)
	drainSynchronizedUI()
	if !reflect.DeepEqual(got, []Appearance{AppearanceDark}) {
		t.Fatalf("notifications = %#v", got)
	}
}
~~~

- [ ] **Step 2: Run the test and verify failure**

Run:

~~~powershell
go test ./internal/ui -run TestUIEnvironmentNotifiesOnceAndStopsAfterDispose -count=1
~~~

Expected: FAIL until lifecycle semantics are complete.

- [ ] **Step 3: Migrate application entry wiring**

Verify the UIEnvironment created in Task 8 is shared by NewMainWindowWithEnvironment, NewTray, ShowAddHostDialog, settings, errors, edits, renames, confirmations, and logs. Remove any compatibility constructor that creates an unmanaged environment.

~~~go
environment, err := ui.NewUIEnvironment()
if err != nil {
	log.Fatal(err)
}
defer environment.Dispose()

mainWindow, err := ui.NewMainWindowWithEnvironment(runtime.model, runtime.manager, environment)
~~~

- [ ] **Step 4: Handle live appearance and DPI updates**

On appearance notification, update the DWM title-bar attribute for every open top-level window, call SetTheme with DarkMode_Explorer or Explorer on compatible native controls, rebuild UIResources, and invalidate custom widgets. On Walk DPI application, recompute Metrics, row heights, card padding, and fonts, then request layout.

- [ ] **Step 5: Verify disposal ordering**

Explicit Quit stops config watching and tunnel activity as before. Close log refresh contexts before log widgets. Dispose tray before main window, owned windows before UIEnvironment, and UIEnvironment before walk application teardown.

Delete text_scale.go after confirming no ApplyStandardTextScale caller remains.

- [ ] **Step 6: Run integration verification**

Run:

~~~powershell
go test ./... -count=1
go vet ./...
~~~

Expected: both commands succeed.

- [ ] **Step 7: Commit**

~~~powershell
git add Windows/cmd/tunneldock/main.go Windows/internal/ui/main_window.go Windows/internal/ui/tray.go Windows/internal/ui/environment.go Windows/internal/ui/ui_smoke_windows_test.go Windows/internal/ui/text_scale.go
git commit -m "feat(windows): integrate adaptive UI lifecycle"
~~~

---

### Task 12: Visual acceptance, documentation, and release build

**Files:**
- Modify: Windows/docs/manual-acceptance.md
- Modify: Windows/README.md only if the visible behavior documentation requires clarification.

**Interfaces:**
- Consumes: completed Windows UI and existing release script.
- Produces: verified Windows/dist/TunnelDock.exe and recorded manual acceptance results.

- [ ] **Step 1: Extend the manual acceptance matrix**

Add explicit checks for:

~~~text
Light theme: main window plus every dialog
Dark theme: main window plus every dialog
Display scale: 100%, 150%, 200%
Rows: empty, single, multiple, long names, inline errors
States: disconnected, connecting, connected, reconnecting, failed
Sidebar: All Tunnels, available, configuration error, missing, search
Input: mouse, wheel, Tab, Shift-Tab, arrows, Enter, Space, Esc
Refresh: selection, focus, scroll, and Advanced disclosure preserved
Theme change: every open window updates without restart
~~~

- [ ] **Step 2: Run formatting and all automated verification**

Run from Windows:

~~~powershell
gofmt -w internal/ui cmd/tunneldock
go test ./... -count=1
go vet ./...
.\scripts\build.ps1
git diff --check
git status --short
~~~

Expected: tests and vet pass; build creates dist\TunnelDock.exe; diff check is silent; only intentional source/document changes are present.

- [ ] **Step 3: Launch the exact release artifact**

Run:

~~~powershell
Start-Process -FilePath .\dist\TunnelDock.exe
~~~

Confirm in Task Manager that the running image path is the newly built Windows\dist\TunnelDock.exe.

- [ ] **Step 4: Complete the visual matrix**

Compare the Windows app against the macOS implementation for hierarchy, spacing, selection treatment, inline tunnel actions, Quick Forward, dialogs, and logs. Record each light/dark and DPI result in Windows/docs/manual-acceptance.md. Fix any failed item, rerun the targeted test, rebuild, and repeat that row of the matrix.

- [ ] **Step 5: Confirm platform isolation**

Run from the worktree root:

~~~powershell
git diff --name-only c439749..HEAD
~~~

Expected: implementation files are under Windows plus the approved docs/superpowers design and plan documents; no Sources/TunnelDock macOS file appears.

- [ ] **Step 6: Commit**

~~~powershell
git add Windows/docs/manual-acceptance.md Windows/README.md
git commit -m "docs(windows): record redesigned UI acceptance"
~~~

- [ ] **Step 7: Final status evidence**

Run:

~~~powershell
git status --short --branch
git log -12 --oneline --decorate
~~~

Expected: windows-v1 is clean and the UI redesign commits are visible in task order.
