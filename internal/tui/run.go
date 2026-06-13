package tui

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// Run is the application entry point: it loads (or bootstraps) config,
// resolves the theme, authenticates the active instance, and runs the TUI.
func Run() error {
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
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		_ = api.Logout(ctx) // best-effort: don't leak a session seat
	}()

	m := New(cfg, path, api, th)
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
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
