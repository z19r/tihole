package core

import "charm.land/bubbles/v2/key"

// KeyMap holds the global key bindings available on every screen. Screen-local
// bindings are returned from Screen.Help() and merged into the help bar.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Back        key.Binding
	Palette     key.Binding
	SwitchInst  key.Binding
	ToggleBlock key.Binding
	CycleTheme  key.Binding
	Refresh     key.Binding
	Splash      key.Binding
	Help        key.Binding
	Quit        key.Binding
}

// DefaultKeyMap returns the global bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Palette: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "palette"),
		),
		SwitchInst: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch instance"),
		),
		ToggleBlock: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "toggle blocking"),
		),
		CycleTheme: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "theme"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Splash: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "splash"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp implements help.KeyMap for the compact help bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Palette, k.SwitchInst, k.ToggleBlock, k.CycleTheme, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Enter, k.Back},
		{k.Palette, k.SwitchInst, k.ToggleBlock, k.CycleTheme, k.Refresh, k.Splash},
		{k.Help, k.Quit},
	}
}
