// Package config handles loading, saving, and validating tihole's
// configuration file (~/.config/tihole/config.yaml, mode 0600) and the
// first-run setup wizard.
package config

import (
	"os"
	"path/filepath"
)

// Config is the top-level configuration, matching config.yaml.
type Config struct {
	Active    string         `yaml:"active"`
	Theme     string         `yaml:"theme,omitempty"`
	Refresh   map[string]int `yaml:"refresh,omitempty"`
	Instances []Instance     `yaml:"instances"`
}

// Instance describes a single Pi-hole instance.
//
// VerifyTLS is a *bool so that an omitted `verify_tls` key defaults to true
// (verify enabled) rather than the Go zero value of false. Use VerifyTLSValue
// to read the effective value.
type Instance struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Password    string `yaml:"password,omitempty"`
	PasswordEnv string `yaml:"password_env,omitempty"`
	VerifyTLS   *bool  `yaml:"verify_tls,omitempty"`
}

// VerifyTLSValue reports the effective TLS-verification setting. When the
// `verify_tls` key is absent (nil pointer) it defaults to true.
func (i *Instance) VerifyTLSValue() bool {
	if i.VerifyTLS == nil {
		return true
	}
	return *i.VerifyTLS
}

// DefaultPath returns the default config path:
// <UserConfigDir>/tihole/config.yaml.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tihole", "config.yaml"), nil
}
