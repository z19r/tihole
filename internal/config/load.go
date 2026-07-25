package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configFileMode = 0o600
	configDirMode  = 0o700
)

// Load reads, parses, and validates the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	if err := Validate(&c); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return &c, nil
}

// Save marshals c to YAML and writes it to path with mode 0600, creating the
// parent directory (mode 0700) if needed.
func Save(path string, c *Config) error {
	if c == nil {
		return fmt.Errorf("cannot save nil config")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, configDirMode); err != nil {
		return fmt.Errorf("create config dir %q: %w", dir, err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}

	// WriteFile only applies the mode on creation; enforce it explicitly so an
	// existing file with looser permissions is tightened to 0600.
	if err := os.Chmod(path, configFileMode); err != nil {
		return fmt.Errorf("chmod config %q: %w", path, err)
	}

	return nil
}

// ResolvePassword returns the instance's usable password: the inline Password
// if set, otherwise the value of the PasswordEnv environment variable. It
// errors when neither source yields a secret.
func (i *Instance) ResolvePassword() (string, error) {
	if i.Password != "" {
		return i.Password, nil
	}
	if i.PasswordEnv != "" {
		if v := os.Getenv(i.PasswordEnv); v != "" {
			return v, nil
		}
		return "", fmt.Errorf(
			"password_env %q is empty or unset",
			i.PasswordEnv,
		)
	}
	return "", fmt.Errorf("instance %q has no password or password_env", i.Name)
}
