package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Validate checks a Config for structural correctness:
//   - at least one instance is configured
//   - each instance has a non-empty, unique name
//   - each instance URL parses as an absolute http/https URL
//   - `active` names exactly one existing instance
//
// All problems are aggregated into a single error.
func Validate(c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}

	var errs []error

	if len(c.Instances) == 0 {
		errs = append(errs, errors.New("no instances configured"))
	}

	seen := make(map[string]bool, len(c.Instances))
	for idx, inst := range c.Instances {
		name := strings.TrimSpace(inst.Name)
		if name == "" {
			errs = append(errs, fmt.Errorf("instance %d: name is required", idx))
		} else if seen[name] {
			errs = append(errs, fmt.Errorf("instance %q: duplicate name", name))
		} else {
			seen[name] = true
		}

		if err := validateURL(inst.URL); err != nil {
			errs = append(errs, fmt.Errorf("instance %q: %w", displayName(name, idx), err))
		}
	}

	if strings.TrimSpace(c.Active) == "" {
		errs = append(errs, errors.New("active instance is required"))
	} else if len(c.Instances) > 0 && !seen[c.Active] {
		errs = append(errs, fmt.Errorf("active %q does not match any instance", c.Active))
	}

	return errors.Join(errs...)
}

func displayName(name string, idx int) string {
	if name == "" {
		return fmt.Sprintf("#%d", idx)
	}
	return name
}

func validateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("url %q has no host", raw)
	}
	return nil
}
