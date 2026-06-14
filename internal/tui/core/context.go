package core

import (
	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// AppContext is the shared, injected dependency set every screen receives at
// construction. No package-level globals. Theme is a pointer so a live
// re-theme is visible everywhere immediately.
type AppContext struct {
	// API is the active instance's client. Swapped on instance switch.
	API *pihole.Client
	// Theme is held behind a pointer so SetThemeMsg re-theme is instant.
	Theme *theme.Theme
	// Keys is the global key map.
	Keys KeyMap
	// InstanceName is the name of the currently active instance.
	InstanceName string
	// Config is the loaded configuration (held behind a pointer so the
	// Settings screen and root see the same instance list). Nil in unit tests
	// that don't exercise config-backed screens.
	Config *config.Config
	// ConfigPath is where Config is persisted (0600). Empty in such tests.
	ConfigPath string
}

// InstancesChangedMsg is emitted by Settings → Connections after the instance
// list is edited and persisted. The root swaps in the new config and, if the
// active instance was removed, activates another.
type InstancesChangedMsg struct {
	Config *config.Config
}

// SetThemeMsg requests a live re-theme across all screens.
type SetThemeMsg struct {
	Theme *theme.Theme
}

// SwitchInstanceMsg requests activating a different configured instance.
type SwitchInstanceMsg struct {
	Name string
}

// ErrorMsg carries a non-fatal error to be surfaced as an inline banner/toast.
type ErrorMsg struct {
	Screen string
	Err    error
}
