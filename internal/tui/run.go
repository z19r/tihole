package tui

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// Run is the application entry point: it loads (or bootstraps) config, resolves
// the theme, and runs the TUI starting on the dashboard.
func Run() error {
	return run(core.PageDashboard)
}

// RunConfig runs the TUI starting on the Settings/config editor. It is the
// entry point for `tihole config`, so a broken connection can be fixed in-app.
func RunConfig() error {
	return run(core.PageSettings)
}

// run loads config, resolves the theme, builds a client, and runs the TUI on
// the given start page. A wrong address or failed auth never aborts startup:
// the client authenticates lazily and reports failures as in-app banners, and a
// structurally broken instance drops the user into the config editor.
func run(start core.PageID) error {
	path, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := loadOrBootstrap(path)
	if err != nil {
		return err
	}

	th, err := theme.Resolve(cfg.Theme)
	if err != nil {
		// Resolve already falls back to a built-in theme; keep going.
		fmt.Fprintf(os.Stderr, "theme %q: %v (using fallback)\n", cfg.Theme, err)
	}

	api, err := clientFor(cfg, cfg.Active)
	if err != nil {
		// Don't die on a broken instance: open the config editor so the user
		// can fix it, backed by a best-effort client for the status bar.
		fmt.Fprintf(os.Stderr, "tihole: %v — opening config editor\n", err)
		api = fallbackClient(cfg, cfg.Active)
		start = core.PageSettings
	}
	defer logout(api)

	m := New(cfg, path, api, th)
	m.active = start
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}

// logout best-effort ends the session on exit so we don't leak a seat.
func logout(api *pihole.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	_ = api.Logout(ctx)
}

// loadOrBootstrap loads the config file, or runs the first-run wizard if it
// doesn't exist yet.
func loadOrBootstrap(path string) (*config.Config, error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		cfg, err := config.RunFirstRun(path)
		if err != nil {
			return nil, fmt.Errorf("first-run setup: %w", err)
		}
		return cfg, nil
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}
	return cfg, nil
}
