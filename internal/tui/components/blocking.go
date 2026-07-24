package components

// BlockState describes the current global blocking status. The root tracks it
// for the quick-toggle shortcut ("d"); the Blocking screen renders the full
// control with its duration options.
type BlockState struct {
	// Known is false until the first fetch resolves.
	Known bool
	// Enabled is true when blocking is active.
	Enabled bool
	// CountdownSecs, when > 0, shows a temporary-disable countdown.
	CountdownSecs int
}
