package core

import (
	"context"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
)

// Source is the set of combined-capable calls; *pihole.Client and
// *pihole.Fleet both satisfy it.
type Source interface {
	Summary(context.Context) (pihole.Summary, error)
	History(context.Context) (pihole.History, error)
	QueryTypes(context.Context) (pihole.QueryTypes, error)
	Upstreams(context.Context) (pihole.Upstreams, error)
	TopDomains(context.Context, bool, int) (pihole.TopDomains, error)
	TopClients(context.Context, bool, int) (pihole.TopClients, error)
	Queries(context.Context, pihole.QueryFilter) (pihole.QueriesPage, error)
	Blocking(context.Context) (pihole.BlockingStatus, error)
	SetBlocking(context.Context, bool, *float64) error
}

var (
	_ Source = (*pihole.Client)(nil)
	_ Source = (*pihole.Fleet)(nil)
)

// AppContext is the shared, injected dependency set every screen receives at
// construction. No package-level globals. Theme is a pointer so a live
// re-theme is visible everywhere immediately.
type AppContext struct {
	// API is the active instance's client. Swapped on instance switch.
	API *pihole.Client
	// Fleet fans reads out across every instance; nil when not built.
	Fleet *pihole.Fleet
	// Combined is the user's combined-stats preference.
	Combined bool
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

// CombinedActive reports whether reads should go through the fleet.
func (c *AppContext) CombinedActive() bool {
	return c.Combined && c.Fleet != nil && c.Fleet.Len() >= 2
}

// Source returns the fleet when combined mode is active, else the API.
func (c *AppContext) Source() Source {
	if c.CombinedActive() {
		return c.Fleet
	}
	return c.API
}
