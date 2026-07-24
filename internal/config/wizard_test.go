package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestVerifyTLSDefaultsToTrueWhenKeyOmitted(t *testing.T) {
	// Arrange: YAML without a verify_tls key.
	raw := []byte(`
active: home
instances:
  - name: home
    url: https://192.168.1.2
    password: secret
`)

	// Act
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Assert
	inst := c.Instances[0]
	if inst.VerifyTLS != nil {
		t.Fatalf("expected nil pointer for omitted verify_tls, got %v", *inst.VerifyTLS)
	}
	if !inst.VerifyTLSValue() {
		t.Fatal("omitted verify_tls should default to true")
	}
}

func TestVerifyTLSHonorsExplicitFalse(t *testing.T) {
	// Arrange
	raw := []byte(`
active: home
instances:
  - name: home
    url: https://192.168.1.2
    password: secret
    verify_tls: false
`)

	// Act
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Assert
	if c.Instances[0].VerifyTLSValue() != false {
		t.Fatal("explicit verify_tls: false should be honored")
	}
}

func TestBuildConfigProducesActiveSingleInstance(t *testing.T) {
	// Arrange
	in := wizardInputs{Name: " home ", URL: " https://192.168.1.2 ", Password: "secret"}

	// Act
	c, err := buildConfig(in)

	// Assert
	if err != nil {
		t.Fatalf("buildConfig error: %v", err)
	}
	if c.Active != "home" {
		t.Fatalf("active = %q, want home (trimmed)", c.Active)
	}
	if len(c.Instances) != 1 || c.Instances[0].URL != "https://192.168.1.2" {
		t.Fatalf("instance not built correctly: %+v", c.Instances)
	}
}

func TestBuildConfigRejectsInvalidInputs(t *testing.T) {
	// Arrange: empty name and bad URL.
	in := wizardInputs{Name: "", URL: "not-a-url", Password: "x"}

	// Act
	_, err := buildConfig(in)

	// Assert
	if err == nil {
		t.Fatal("expected validation error from buildConfig, got nil")
	}
}

func TestFinishWritesConfigAndInvokesValidateFunc(t *testing.T) {
	// Arrange
	var gotInst Instance
	fr := &FirstRun{ValidateFunc: func(inst Instance) error {
		gotInst = inst
		return nil
	}}
	in := wizardInputs{Name: "home", URL: "https://192.168.1.2", Password: "secret"}
	path := filepath.Join(t.TempDir(), "config.yaml")

	// Act
	c, err := fr.finish(in, path)

	// Assert
	if err != nil {
		t.Fatalf("finish error: %v", err)
	}
	if gotInst.Name != "home" {
		t.Fatalf("ValidateFunc got instance %+v", gotInst)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("config not written: %v", statErr)
	}
	if c.Active != "home" {
		t.Fatalf("active = %q", c.Active)
	}
}

func TestFinishPropagatesValidateFuncError(t *testing.T) {
	// Arrange
	fr := &FirstRun{ValidateFunc: func(inst Instance) error {
		return os.ErrPermission
	}}
	in := wizardInputs{Name: "home", URL: "https://192.168.1.2", Password: "secret"}
	path := filepath.Join(t.TempDir(), "config.yaml")

	// Act
	_, err := fr.finish(in, path)

	// Assert
	if err == nil {
		t.Fatal("expected auth-validation error")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("config should not be written when validation fails")
	}
}

func TestFinishRejectsInvalidInputs(t *testing.T) {
	// Arrange
	fr := &FirstRun{}
	in := wizardInputs{Name: "", URL: "bad", Password: "x"}

	// Act
	_, err := fr.finish(in, filepath.Join(t.TempDir(), "c.yaml"))

	// Assert
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBuildConfigResultIsSavable(t *testing.T) {
	// Arrange
	in := wizardInputs{Name: "home", URL: "https://192.168.1.2", Password: "secret"}
	c, err := buildConfig(in)
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")

	// Act
	if err := Save(path, c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Assert
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}
