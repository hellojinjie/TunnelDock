# TunnelDock Windows UI Redesign

**Date:** 2026-08-28

**Status:** Approved in conversation; awaiting written-spec review

## Goal

Reimplement the complete TunnelDock Windows interface so that its information hierarchy, component structure, spacing, and interaction model closely follow the existing macOS application. Replace the current table-oriented presentation with purpose-built components, especially for the host sidebar and tunnel rows.

The redesign covers the main window and every secondary window or dialog. It supports both light and dark system themes. The Windows title bar and operating-system window behavior remain native; the application content follows the macOS design language.

## Scope and invariants

This work changes the Windows view layer under `Windows/internal/ui` and adds focused, UI-facing presentation helpers where needed. It must not change:

- the macOS source tree;
- the persisted tunnel schema;
- SSH discovery, process, security, or lifecycle behavior;
- temporary-versus-saved tunnel semantics;
- the existing application and tunnel-manager ownership boundaries.

The existing Windows application model and tunnel manager remain the sources of truth. Walk remains the UI toolkit. No WebView, HTML UI, Electron, Wails, Fyne, or CGO dependency is introduced.

## Chosen approach

Use a hybrid custom-component architecture:

- render host and tunnel rows with DPI-aware Walk custom widgets;
- place rows in Walk scroll containers rather than `TableView` or `ListView`;
- retain native text fields, combo boxes, menus, and window behavior for input-method support, keyboard handling, and accessibility;
- wrap native controls in consistent custom layouts and themed surfaces;
- share one theme, metrics, font, and icon system across the main window and all dialogs.

A fully canvas-rendered application was rejected because it would require reimplementing text input, focus, keyboard navigation, and accessibility. Owner-drawing the existing table controls was rejected because their column and selection model still prevents the macOS-style row hierarchy and inline actions.

## UI architecture

### Theme manager

A central theme manager owns semantic design tokens rather than scattering literal colors and dimensions through widgets. It provides:

- light and dark palettes;
- background, sidebar, elevated-surface, border, primary-text, secondary-text, disabled, accent, success, warning, and error colors;
- typography for large titles, section titles, body text, captions, buttons, and log text;
- logical spacing, corner-radius, icon-size, row-height, and focus-ring metrics;
- DPI-scaled brushes, pens, fonts, and icons;
- notification when the Windows system theme changes.

Theme changes update the native title-bar theme where supported, apply compatible themes to native child controls, and invalidate custom widgets. DPI changes rebuild scaled resources and request layout without restarting the process.

All disposable GDI and Walk resources have explicit ownership. Replacing a theme or DPI resource set disposes the superseded resources only after widgets stop referencing it.

### Main window shell

The main window keeps a native Windows frame and uses a two-pane body:

- resizable default size approximately 1120 by 760 logical pixels;
- minimum size approximately 900 by 600 logical pixels;
- sidebar ideal width 240, constrained to approximately 210 through 320;
- detail content scrollable with 24-pixel logical margins and a content width near the macOS view's 760-pixel maximum.

`MainWindow` coordinates page selection and model updates. It does not own the internal rendering details of every child. Its principal components are:

- `Sidebar`;
- `HostDetailPage` or `AllTunnelsPage`;
- `TunnelList`;
- `QuickForwardCard`;
- a settings toolbar action.

### Sidebar

The sidebar uses custom rows and contains:

1. an All Tunnels row;
2. an SSH Hosts section header with compact search affordance;
3. add-host, open-config, and refresh actions;
4. available/configuration-error host rows;
5. a Missing Hosts section when applicable.

Rows use rounded selection backgrounds, icon-plus-label content, and optional activity or warning indicators. They do not expose table columns, headers, grid lines, or native full-row selection chrome. Search filters aliases and resolved host fields according to the existing application model without reordering the source data.

The selected row remains stable across host refreshes when its alias still exists. If it no longer exists, the application falls back to All Tunnels, matching the current behavior.

### Detail pages and cards

The detail page follows the macOS vertical hierarchy:

- large host or All Tunnels title;
- secondary effective connection text;
- configuration or missing-host status when relevant;
- a Recent Tunnels or All Tunnels card;
- a Quick Forward card for available hosts.

Cards use a subtle semantic surface color, thin border, and rounded corners. They replace traditional Windows group boxes.

## Custom row components

### Host row

Each host row is a reusable custom widget with normal, hover, selected, pressed, focused, and disabled states. It draws an availability icon, alias, and activity or warning indicator. It supports:

- mouse selection;
- Up/Down keyboard navigation within the sidebar;
- Enter or Space activation;
- a visible focus ring;
- tooltip text for non-obvious status icons.

### Tunnel row

Each tunnel row renders a variable-height, macOS-style horizontal layout:

- state icon on the left;
- primary tunnel name;
- secondary `localAddress:localPort -> remoteHost:remotePort` text;
- optional error text on a third line;
- state label;
- Open in Browser icon while connected;
- Connect or Disconnect button;
- More menu button.

The row owns its action hit areas. A user does not need to select a table row and then move to a detached button bar. Hover and pressed feedback applies independently to the row and action regions. Keyboard navigation moves between rows; Tab reaches actionable controls; Enter invokes the primary action when appropriate.

The component computes height from its presentation model. Rows without errors use a compact standard height; error rows expand enough to show the message without overlapping adjacent content. Long names and addresses ellipsize predictably and provide full text through tooltips.

The More menu retains native popup-menu behavior. Its items and enabled states follow the existing rules for temporary, saved, connected, and disconnected tunnels.

## Quick Forward

The Quick Forward card mirrors the macOS arrangement:

- Remote Port is the primary visible input;
- Connect is beside the primary input;
- Advanced is collapsed initially;
- expanded fields are Local Port, Remote Host, Local Address, and Browser URL Scheme;
- validation appears inline below the relevant controls.

Local Port follows Remote Port until the user manually changes Local Port. A port conflict expands Advanced, focuses Local Port, and preserves all other input. While a connection attempt is active, the primary action reflects Connecting state and prevents duplicate submission.

## Dialog and secondary-window system

All dialogs use a shared shell for content margins, background, title typography, form spacing, validation, and bottom action placement. Native modality, ownership, Enter-default, Esc-cancel, Tab traversal, and resizing rules remain intact.

### Add SSH Host and Edit Tunnel

These use a title and short description, a vertically aligned form, inline validation, and right-aligned Cancel and primary buttons. Invalid input keeps the dialog open and focuses the first invalid field. Validation does not dismiss the dialog and then show a separate message box.

### Rename Tunnel

This is a compact single-field dialog. It selects the existing name when opened, accepts with Enter, and cancels with Esc.

### Settings

Settings use card-style rows. The current tray-icon setting is applied immediately and persisted through the existing settings store. The view also communicates that appearance follows the Windows system theme; no separate application theme preference is added in this redesign.

### Tunnel Log

The log viewer is an independently resizable window with a themed header showing tunnel name and state. The log body uses a readable monospace font, supports live updates, scrolling, selection, and copy, and remains read-only. Closing it never changes tunnel state.

### Connection errors and confirmations

Connection errors use a consistent application dialog containing a summary, specific cause, and suggested action. The existing Open SSH Terminal recovery action remains available when applicable.

Deletion uses an application-styled warning confirmation that names the tunnel being deleted. Destructive actions remain unavailable for running tunnels.

## Presentation data and update flow

Business objects are converted into immutable UI presentation models before drawing. Custom paint and hit-test code does not mutate application or tunnel objects.

Host alias is the stable sidebar identity. Tunnel runtime ID is the stable tunnel-row identity. Refresh uses reconciliation rather than resetting entire list controls:

1. obtain a model snapshot;
2. convert it to presentation models;
3. reuse matching row widgets by stable identity;
4. update, insert, remove, or reorder only affected rows;
5. preserve selection, keyboard focus, scroll offset, and disclosure state when still valid;
6. request layout and invalidate only changed regions.

Background configuration refreshes, tunnel state notifications, and connection completions marshal their UI changes through Walk's synchronization boundary.

Inline actions call the same existing manager operations used by the current window. They do not duplicate tunnel logic inside widgets. During asynchronous operations, only relevant controls are disabled and the affected row immediately reflects the transitional state.

Temporary tunnels appear only after a successful Quick Forward connection. Disconnect removes a temporary tunnel; Save converts it to a persisted tunnel; successful saved connections remain in Recent Tunnels. These rules stay consistent with the macOS implementation and current Windows domain behavior.

## Theme behavior

The application follows the current Windows app theme in both the main window and every owned window. It handles system theme-change notifications and updates open windows in place.

Light and dark palettes use semantic contrast rather than color inversion. Status colors retain recognizable success, connecting, warning, and failure meaning in both palettes. Focus indicators remain visible against selected and unselected surfaces.

Custom icons are vector-like drawing primitives or resolution-independent resources where practical. Raster resources include variants or scale from sufficiently large sources without relying on font glyphs such as `+`, `✎`, `↻`, or `▦` as interface icons.

## Error handling and fallback

Widget-construction failures propagate to the application startup boundary after disposing partially constructed resources. Theme or optional icon-loading failures are logged and fall back to a safe semantic palette or simplified drawable icon. A cosmetic failure must not alter SSH security, persistence, or process behavior.

Action errors appear next to their context and, where additional explanation or recovery is necessary, open the unified error dialog. Background work never directly touches UI widgets.

## Testing

### Automated tests

Add unit coverage for:

- light and dark theme-token completeness and contrast invariants;
- DPI metric scaling;
- host and tunnel presentation-model generation;
- variable tunnel-row height;
- hit regions and action routing;
- action visibility and enabled-state rules;
- stable-ID reconciliation and preservation of selection state;
- Quick Forward expanded/error states;
- dialog validation that keeps invalid dialogs open.

Add Windows UI smoke tests where Walk permits reliable construction of hidden windows. These tests create, refresh, theme-switch, DPI-refresh, and dispose the main window and each dialog without requiring real SSH connections.

Before completion run the existing Windows test suite, `go vet ./...`, the resource/build script, and `git diff --check` using the repository's configured Go toolchain. Existing application, persistence, SSH, and tunnel-manager tests must continue to pass.

### Manual visual and interaction matrix

Inspect the main window and every dialog in light and dark modes at 100%, 150%, and 200% display scale. Cover:

- empty, single-row, and multi-row lists;
- long host and tunnel names;
- multiple tunnels and multiple states;
- connecting, reconnecting, connected, disconnected, failed, and inline-error states;
- missing and configuration-error hosts;
- Quick Forward collapsed, expanded, invalid, connecting, and port-conflict states;
- hover, pressed, selected, disabled, and focus visuals;
- mouse wheel, scroll bar, Tab, Shift-Tab, arrow keys, Enter, Space, and Esc;
- Add Host, Edit, Rename, Settings, Log, Error, and Delete dialogs.

Compare the resulting information hierarchy, spacing, row content, inline actions, and Quick Forward structure with the macOS application. Pixel identity of the operating-system title bar is explicitly out of scope.

Build the final GUI executable and start that exact artifact on Windows. A successful unit-test run alone is not sufficient UI acceptance.

## Definition of done

The redesign is complete when:

- no host or tunnel list in the main window uses `TableView` or `ListView`;
- the main window and every secondary window use the shared component and theme system;
- light and dark modes update live and remain legible at supported DPI values;
- tunnel actions are performed directly from their rows;
- selection, focus, scrolling, and disclosure state survive ordinary refreshes;
- keyboard and mouse workflows pass the manual matrix;
- existing Windows behavior and automated tests remain intact;
- the Windows GUI executable builds and is visually inspected;
- macOS source and behavior are unchanged.
