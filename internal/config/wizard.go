package config

import (
	"fmt"
	"strings"

	"charm.land/huh/v2"
)

// wizardInputs holds the raw values collected from the first-run form.
type wizardInputs struct {
	Name     string
	URL      string
	Password string
}

// buildConfig turns collected wizard inputs into a validated Config with the
// single instance marked active. It is the pure, testable core of the wizard.
func buildConfig(in wizardInputs) (*Config, error) {
	name := strings.TrimSpace(in.Name)
	inst := Instance{
		Name:     name,
		URL:      strings.TrimSpace(in.URL),
		Password: in.Password,
	}
	c := &Config{
		Active:    name,
		Instances: []Instance{inst},
	}
	if err := Validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// RunFirstRun presents the interactive first-run wizard (name, URL, password),
// builds a Config with that single instance active, and writes it to path.
//
// ValidateFunc, when set, is invoked with the assembled Instance so callers can
// inject auth-validation (e.g. attempt a login). A nil ValidateFunc skips that
// step.
type FirstRun struct {
	ValidateFunc func(inst Instance) error
}

// RunFirstRun is a convenience wrapper around FirstRun with no injected
// validation.
func RunFirstRun(path string) (*Config, error) {
	return (&FirstRun{}).Run(path)
}

// Run executes the wizard and writes the resulting config to path.
func (f *FirstRun) Run(path string) (*Config, error) {
	var in wizardInputs

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Instance name").
				Placeholder("home").
				Value(&in.Name).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Pi-hole URL").
				Placeholder("https://192.168.1.2").
				Value(&in.URL).
				Validate(func(s string) error { return validateURL(s) }),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&in.Password).
				Validate(huh.ValidateNotEmpty()),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("first-run wizard: %w", err)
	}

	return f.finish(in, path)
}

// finish assembles, optionally auth-validates, and persists the config from
// collected inputs. It is the pure, testable tail of the wizard (everything
// after the interactive form).
func (f *FirstRun) finish(in wizardInputs, path string) (*Config, error) {
	c, err := buildConfig(in)
	if err != nil {
		return nil, err
	}

	if f.ValidateFunc != nil {
		if err := f.ValidateFunc(c.Instances[0]); err != nil {
			return nil, fmt.Errorf("auth validation failed: %w", err)
		}
	}

	if err := Save(path, c); err != nil {
		return nil, err
	}

	return c, nil
}
