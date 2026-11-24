package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	c.Active = strings.TrimSpace(c.Active)
	for i := range c.Instances {
		c.Instances[i].Name = strings.TrimSpace(c.Instances[i].Name)
	}

	if err := Validate(&c); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return &c, nil
}

// Save marshals c to YAML and atomically writes it to path with mode 0600,
// creating the parent directory (mode 0700) if needed.
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

	return writeAtomic(path, data)
}

// writeAtomic writes data to a 0600 temp file beside path, syncs it, and
// renames it over path, so a failed write never truncates the original.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(
		tmp,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		configFileMode,
	)
	if err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	_, err = f.Write(data)
	if err == nil {
		// OpenFile only applies the mode on creation.
		err = f.Chmod(configFileMode)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write config %q: %w", path, err)
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
