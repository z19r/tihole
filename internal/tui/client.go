package tui

import (
	"fmt"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// clientFor builds a pihole.Client for the named instance WITHOUT
// authenticating. The client's do() logs in transparently on the first request,
// so a wrong address or bad password surfaces as an in-app error banner rather
// than crashing at startup. An error is returned only for structural config
// problems the caller should report (unknown instance, unresolvable password).
func clientFor(cfg *config.Config, name string) (*pihole.Client, error) {
	inst := findInstance(cfg, name)
	if inst == nil {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	password, err := inst.ResolvePassword()
	if err != nil {
		return nil, fmt.Errorf("instance %q: %w", name, err)
	}

	opts := []pihole.Option{
		pihole.WithInsecureTLS(!inst.VerifyTLSValue()),
	}
	return pihole.New(inst.URL, password, opts...), nil
}

// fallbackClient builds a best-effort, unauthenticated client so the TUI can
// still start (in the config editor) when clientFor fails. It never returns nil.
func fallbackClient(cfg *config.Config, name string) *pihole.Client {
	if inst := findInstance(cfg, name); inst != nil {
		return pihole.New(inst.URL, "", pihole.WithInsecureTLS(!inst.VerifyTLSValue()))
	}
	return pihole.New("", "")
}

func findInstance(cfg *config.Config, name string) *config.Instance {
	for i := range cfg.Instances {
		if cfg.Instances[i].Name == name {
			return &cfg.Instances[i]
		}
	}
	return nil
}
