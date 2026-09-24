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

func TestPortalCheckIntervalDefault(t *testing.T) {
	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 30 {
		t.Fatalf("PortalCheckInterval = %d, want 30", cfg.PortalCheckInterval)
	}
}

func TestPortalCheckIntervalFlag(t *testing.T) {
	cfg, err := Load([]string{"--portal-check-interval=15"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 15 {
		t.Fatalf("PortalCheckInterval = %d, want 15", cfg.PortalCheckInterval)
	}
}

func TestPortalCheckIntervalEnv(t *testing.T) {
	os.Setenv("NM_WEBUI_PORTAL_CHECK_INTERVAL", "45")
	defer os.Unsetenv("NM_WEBUI_PORTAL_CHECK_INTERVAL")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 45 {
		t.Fatalf("PortalCheckInterval = %d, want 45", cfg.PortalCheckInterval)
	}
}

func TestPortalCheckIntervalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("portal-check-interval: 60\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 60 {
		t.Fatalf("PortalCheckInterval = %d, want 60", cfg.PortalCheckInterval)
	}
}

func TestPortalCheckIntervalFlagOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("portal-check-interval: 60\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path, "--portal-check-interval=5"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 5 {
		t.Fatalf("PortalCheckInterval = %d, want 5 (flag wins over file)", cfg.PortalCheckInterval)
	}
}

func TestPortalCheckIntervalClampedPositive(t *testing.T) {
	os.Setenv("NM_WEBUI_PORTAL_CHECK_INTERVAL", "0")
	defer os.Unsetenv("NM_WEBUI_PORTAL_CHECK_INTERVAL")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalCheckInterval != 30 {
		t.Fatalf("PortalCheckInterval = %d, want clamped 30", cfg.PortalCheckInterval)
	}
}

func TestPortalProxyListenDefault(t *testing.T) {
	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "0.0.0.0:8091" {
		t.Fatalf("PortalProxyListen = %q, want 0.0.0.0:8091", cfg.PortalProxyListen)
	}
	if cfg.PortalProxyPort() != "8091" {
		t.Fatalf("PortalProxyPort = %q, want 8091", cfg.PortalProxyPort())
	}
	if cfg.ListenPort() != "8090" {
		t.Fatalf("ListenPort = %q, want 8090", cfg.ListenPort())
	}
	if cfg.PortalProxyOffset() != 1 {
		t.Fatalf("PortalProxyOffset = %d, want 1", cfg.PortalProxyOffset())
	}
}

func TestPortalProxyListenFlag(t *testing.T) {
	cfg, err := Load([]string{"--portal-proxy-listen=127.0.0.1:9091"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "127.0.0.1:9091" {
		t.Fatalf("PortalProxyListen = %q, want 127.0.0.1:9091", cfg.PortalProxyListen)
	}
	if cfg.PortalProxyPort() != "9091" {
		t.Fatalf("PortalProxyPort = %q, want 9091", cfg.PortalProxyPort())
	}
	if cfg.PortalProxyOffset() != 1001 {
		t.Fatalf("PortalProxyOffset = %d, want 1001", cfg.PortalProxyOffset())
	}
}

func TestPortalProxyListenDisable(t *testing.T) {
	cfg, err := Load([]string{"--portal-proxy-listen="})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "" {
		t.Fatalf("PortalProxyListen = %q, want empty (disabled)", cfg.PortalProxyListen)
	}
	if cfg.PortalProxyPort() != "" {
		t.Fatalf("PortalProxyPort = %q, want empty when disabled", cfg.PortalProxyPort())
	}
	if cfg.PortalProxyOffset() != 0 {
		t.Fatalf("PortalProxyOffset = %d, want 0 when disabled", cfg.PortalProxyOffset())
	}
}

func TestPortalProxyListenEnv(t *testing.T) {
	os.Setenv("NM_WEBUI_PORTAL_PROXY_LISTEN", "0.0.0.0:9091")
	defer os.Unsetenv("NM_WEBUI_PORTAL_PROXY_LISTEN")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "0.0.0.0:9091" {
		t.Fatalf("PortalProxyListen = %q, want 0.0.0.0:9091", cfg.PortalProxyListen)
	}
}

func TestPortalProxyListenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("portal-proxy-listen: 0.0.0.0:9095\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "0.0.0.0:9095" {
		t.Fatalf("PortalProxyListen = %q, want 0.0.0.0:9095", cfg.PortalProxyListen)
	}
}

func TestPortalProxyListenFlagOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("portal-proxy-listen: 0.0.0.0:9095\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path, "--portal-proxy-listen=127.0.0.1:9099"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyListen != "127.0.0.1:9099" {
		t.Fatalf("PortalProxyListen = %q, want 127.0.0.1:9099 (flag wins over file)", cfg.PortalProxyListen)
	}
}

func TestPortalProxyListenUnparseablePort(t *testing.T) {
	cfg, err := Load([]string{"--portal-proxy-listen=not-an-addr", "--listen=also-bad"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PortalProxyPort() != "" {
		t.Fatalf("PortalProxyPort = %q, want empty for unparseable address", cfg.PortalProxyPort())
	}
	if cfg.ListenPort() != "8090" {
		t.Fatalf("ListenPort = %q, want fallback 8090", cfg.ListenPort())
	}
}
