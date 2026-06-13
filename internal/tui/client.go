package tui

import (
	"context"
	"fmt"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// clientFor builds a pihole.Client for the named instance and authenticates it.
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
	client := pihole.New(inst.URL, password, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	if err := client.Login(ctx); err != nil {
		return nil, fmt.Errorf("instance %q: authentication failed: %w", name, err)
	}
	return client, nil
}

func findInstance(cfg *config.Config, name string) *config.Instance {
	for i := range cfg.Instances {
		if cfg.Instances[i].Name == name {
			return &cfg.Instances[i]
		}
	}
	return nil
}
