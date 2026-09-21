package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPowerActionPasswordFlag(t *testing.T) {
	cfg, err := Load([]string{"--power-action-password=secret"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.PowerActionEnabled() {
		t.Fatal("PowerActionEnabled() = false, want true")
	}
	if cfg.PowerActionPass != "secret" {
		t.Fatalf("PowerActionPass = %q, want %q", cfg.PowerActionPass, "secret")
	}
}

func TestPowerActionPasswordDisabledByDefault(t *testing.T) {
	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PowerActionEnabled() {
		t.Fatal("PowerActionEnabled() = true by default, want false")
	}
}

func TestPowerActionPasswordEnv(t *testing.T) {
	os.Setenv("NM_WEBUI_POWER_ACTION_PASSWORD", "envsecret")
	defer os.Unsetenv("NM_WEBUI_POWER_ACTION_PASSWORD")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.PowerActionEnabled() || cfg.PowerActionPass != "envsecret" {
		t.Fatalf("env not applied: enabled=%v pass=%q", cfg.PowerActionEnabled(), cfg.PowerActionPass)
	}
}

func TestPowerActionPasswordConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("power-action-password: yamlsecret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.PowerActionEnabled() || cfg.PowerActionPass != "yamlsecret" {
		t.Fatalf("yaml not applied: enabled=%v pass=%q", cfg.PowerActionEnabled(), cfg.PowerActionPass)
	}
}

func TestPowerActionPasswordPrecedenceFlagOverEnv(t *testing.T) {
	os.Setenv("NM_WEBUI_POWER_ACTION_PASSWORD", "envsecret")
	defer os.Unsetenv("NM_WEBUI_POWER_ACTION_PASSWORD")

	cfg, err := Load([]string{"--power-action-password=flagsecret"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PowerActionPass != "flagsecret" {
		t.Fatalf("PowerActionPass = %q, want flagsecret (flag must win over env)", cfg.PowerActionPass)
	}
}

func TestPowerActionPasswordPrecedenceFlagOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("power-action-password: yamlsecret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path, "--power-action-password=flagsecret"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PowerActionPass != "flagsecret" {
		t.Fatalf("PowerActionPass = %q, want flagsecret (flag must win over file)", cfg.PowerActionPass)
	}
}
