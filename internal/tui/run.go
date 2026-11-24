package tui

import (
	"context"
	"fmt"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
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
		fmt.Fprintf(
			os.Stderr,
			"theme %q: %v (using fallback)\n",
			cfg.Theme,
			err,
		)
	}

	api, err := clientFor(cfg, cfg.Active)
	if err != nil {
		// Don't die on a broken instance: open the config editor so the user
		// can fix it, backed by a best-effort client for the status bar.
		fmt.Fprintf(os.Stderr, "tihole: %v — opening config editor\n", err)
		api = fallbackClient(cfg, cfg.Active)
		start = core.PageSettings
	}

	m := New(cfg, path, api, th)
	m.active = start
	p := tea.NewProgram(m)
	_, err = p.Run()
	clients := []*pihole.Client{api, m.ctx.API}
	for _, mem := range m.ctx.Fleet.Members() {
		clients = append(clients, mem.Client)
	}
	m.logouts.flush(clients...)
	return err
}

// logouts remembers clients dropped from the fleet. Each drop is closed by a
// background cmd; exit closes them all again, which Close makes a no-op (or a
// wait) for the ones that already ran, so drops never leak a session.
type logouts struct {
	mu      sync.Mutex
	dropped []*pihole.Client
}

// cmd records cs and returns a cmd that closes them, or nil when cs is empty.
func (l *logouts) cmd(cs []*pihole.Client) tea.Cmd {
	if len(cs) == 0 {
		return nil
	}
	l.mu.Lock()
	l.dropped = append(l.dropped, cs...)
	l.mu.Unlock()
	return func() tea.Msg {
		logoutAll(cs...)
		return nil
	}
}

// flush closes live plus every dropped client once, concurrently, and waits.
func (l *logouts) flush(live ...*pihole.Client) {
	l.mu.Lock()
	all := append(live, l.dropped...)
	l.mu.Unlock()
	logoutAll(all...)
}

// logoutAll best-effort ends each distinct client's session concurrently so we
// don't leak seats, and closes the client so it can't log in again. Errors
// are ignored.
func logoutAll(clients ...*pihole.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), logoutTimeout)
	defer cancel()
	seen := map[*pihole.Client]bool{}
	var wg sync.WaitGroup
	for _, c := range clients {
		if c == nil || seen[c] {
			continue
		}
		seen[c] = true
		wg.Go(func() { _ = c.Close(ctx) })
	}
	wg.Wait()
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
