package core

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// PageID identifies a top-level screen in the sidebar.
type PageID int

const (
	PageDashboard PageID = iota
	PageBlocking
	PageQueryLog
	PageDomains
	PageGroups
	PageClients
	PageAdlists
	PageLocalDNS
	PageSettings
	PageSystem
)

// PageMeta describes a navigable page for the sidebar and router.
type PageMeta struct {
	ID    PageID
	Title string
	Icon  string
}

// pageOrder is the canonical sidebar order.
var pageOrder = []PageMeta{
	{PageDashboard, "Dashboard", "▤"},
	{PageBlocking, "Blocking", "⦸"},
	{PageQueryLog, "Query Log", "≡"},
	{PageDomains, "Domains", "⛒"},
	{PageGroups, "Groups", "⛃"},
	{PageClients, "Clients", "⌂"},
	{PageAdlists, "Adlists", "☰"},
	{PageLocalDNS, "Local DNS", "◈"},
	{PageSettings, "Settings", "⚙"},
	{PageSystem, "System", "❖"},
}

// PageOrder returns the canonical sidebar order.
func PageOrder() []PageMeta { return pageOrder }

// NavigateMsg requests a switch to another page.
type NavigateMsg struct {
	To PageID
}

// Navigate returns a command that emits a NavigateMsg.
func Navigate(to PageID) tea.Cmd {
	return func() tea.Msg { return NavigateMsg{To: to} }
}

// Screen is the contract every content screen implements. It extends the
// Bubble Tea model with lifecycle hooks the root uses to manage focus,
// pollers, and help.
type Screen interface {
	tea.Model
	// Title is shown in the status bar / header.
	Title() string
	// Focus is called when the screen becomes active: start pollers, fetch.
	Focus() tea.Cmd
	// Blur is called when leaving the screen: cancel in-flight work.
	Blur()
	// Help returns the screen-specific key bindings for the help bar.
	Help() []key.Binding
	// SetSize informs the screen of the inner content area (chrome subtracted).
	SetSize(width, height int)
}

// InputCapturer is an optional Screen capability. A screen returns true while it
// has an editable text field focused, so the root delivers raw keys straight to
// it instead of firing global single-key shortcuts (s, d, r, q, j/k, digits, …).
// Screens without text entry simply don't implement it and default to false.
type InputCapturer interface {
	CapturesInput() bool
}

// PanelInteractor is an optional Screen capability. A screen returns false from
// Interactive() when it has no actionable content — a read-only dashboard, for
// instance — so the nav rail refuses to descend into it: enter/tab/→ stay on the
// rail. Screens that don't implement it default to interactive.
type PanelInteractor interface {
	Interactive() bool
}
