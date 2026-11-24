package tui

import (
	"fmt"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
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

// buildFleet groups every configured instance, reusing activeClient for the
// active one so its session is shared. A member of prev whose definition in
// prevCfg is unchanged keeps its client (and session); the clients of prev
// that are no longer used are returned for logout. Instances whose client
// can't be built are recorded as failed.
func buildFleet(
	cfg *config.Config,
	active string,
	activeClient *pihole.Client,
	prev *pihole.Fleet,
	prevCfg *config.Config,
) (*pihole.Fleet, []*pihole.Client) {
	var members []pihole.Member
	failed := map[string]error{}
	used := map[*pihole.Client]bool{activeClient: true}
	for _, inst := range cfg.Instances {
		c := activeClient
		if inst.Name != active {
			c = reusable(prev, prevCfg, inst)
			if c == nil {
				var err error
				if c, err = clientFor(cfg, inst.Name); err != nil {
					failed[inst.Name] = err
					continue
				}
			}
		}
		used[c] = true
		members = append(members, pihole.Member{Name: inst.Name, Client: c})
	}
	var dropped []*pihole.Client
	if prev != nil {
		for _, m := range prev.Members() {
			if !used[m.Client] {
				dropped = append(dropped, m.Client)
			}
		}
	}
	return pihole.NewFleet(members, failed), dropped
}

// reusable returns prev's client for inst when prevCfg defines inst the same
// way, else nil.
func reusable(
	prev *pihole.Fleet,
	prevCfg *config.Config,
	inst config.Instance,
) *pihole.Client {
	if prev == nil || prevCfg == nil {
		return nil
	}
	old := findInstance(prevCfg, inst.Name)
	if old == nil || old.URL != inst.URL || old.Password != inst.Password ||
		old.PasswordEnv != inst.PasswordEnv ||
		old.VerifyTLSValue() != inst.VerifyTLSValue() {
		return nil
	}
	return prev.Client(inst.Name)
}

// fallbackClient builds a best-effort, unauthenticated client so the TUI can
// still start (in the config editor) when clientFor fails. It never returns
// nil.
func fallbackClient(cfg *config.Config, name string) *pihole.Client {
	if inst := findInstance(cfg, name); inst != nil {
		return pihole.New(
			inst.URL,
			"",
			pihole.WithInsecureTLS(!inst.VerifyTLSValue()),
		)
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
