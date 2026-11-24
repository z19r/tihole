package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

	data, err := mergeExisting(path, c)
	if err != nil {
		data, err = yaml.Marshal(c)
	}
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return writeAtomic(path, data)
}

// savedKeys are the top-level keys Config owns; any other key in an existing
// file is left untouched by Save.
var savedKeys = []string{"active", "theme", "instances", "combined"}

// mergeExisting re-encodes the config file at path with c's values written
// into its yaml.Node tree, so hand-written comments and unknown keys survive.
// It errors when there is no existing mapping document to merge into.
func mergeExisting(path string, c *Config) ([]byte, error) {
	old, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(old, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode ||
		doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("not a yaml mapping")
	}
	var fresh yaml.Node
	if err := fresh.Encode(c); err != nil {
		return nil, err
	}
	root := doc.Content[0]
	for _, key := range savedKeys {
		i, j := keyIndex(root, key), keyIndex(&fresh, key)
		switch {
		case j < 0 && i >= 0:
			root.Content = slices.Delete(root.Content, i, i+2)
		case j >= 0 && i >= 0:
			v := fresh.Content[j+1]
			prev := root.Content[i+1]
			v.HeadComment, v.LineComment = prev.HeadComment, prev.LineComment
			v.FootComment = prev.FootComment
			root.Content[i+1] = v
		case j >= 0:
			root.Content = append(root.Content, fresh.Content[j:j+2]...)
		}
	}
	return yaml.Marshal(&doc)
}

// keyIndex returns the index of key's key node in mapping m, or -1.
func keyIndex(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// writeAtomic writes data to a 0600 temp file beside path, syncs it, and
// renames it over path, so a failed write never truncates the original.
func writeAtomic(path string, data []byte) error {
	// Write the link's target so a symlinked config (e.g. dotfiles) survives.
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	// A unique name, so concurrent saves never share a temp file.
	f, err := os.CreateTemp(filepath.Dir(path), "config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	tmp := f.Name()
	err = f.Chmod(configFileMode)
	if err == nil {
		_, err = f.Write(data)
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
