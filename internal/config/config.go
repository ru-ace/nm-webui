package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultInterfaceFilter excludes virtual interfaces (docker/veth/virbr/vbox)
	// and keeps only physical ones by default. "cdc" covers mobile broadband
	// control ports such as cdc-wdm0.
	DefaultInterfaceFilter = `^(eth|en|wlan|wifi|wwan|wwan0|cdc|wl|ra|usb)[0-9A-Za-z.@_-]*$`
)

// ErrHelp is returned when --help is requested.
var ErrHelp = flag.ErrHelp

// Config holds all service options. Order of precedence:
// command line flags > config.yaml > environment variables > defaults.
type Config struct {
	Listen            string
	AuthPass          string
	InterfaceFilter   string
	InterfaceRe       *regexp.Regexp
	TLS               bool
	TLSCert           string
	TLSKey            string
	LogLevel          string
	ConfigFile        string
	ConnectTimeout    int
	CaptivePortalURLs []string
	PortalAllowJS     bool
	PowerActionPass   string
	ShowVersion       bool
}

// Load parses configuration from flags, file and environment.
func Load(args []string) (*Config, error) {
	cfg := &Config{
		Listen:          "0.0.0.0:8080",
		AuthPass:        "",
		InterfaceFilter: DefaultInterfaceFilter,
		TLS:             false,
		TLSCert:         "",
		TLSKey:          "",
		LogLevel:        "info",
		ConnectTimeout:  45,
		PortalAllowJS:   true,
	}

	// 1. Apply environment variables
	if v, ok := os.LookupEnv("NM_WEBUI_LISTEN"); ok {
		cfg.Listen = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_AUTH_PASS"); ok {
		cfg.AuthPass = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_INTERFACE_FILTER"); ok {
		cfg.InterfaceFilter = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_LOG_LEVEL"); ok {
		cfg.LogLevel = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_CONNECT_TIMEOUT"); ok {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.ConnectTimeout = n
		}
	}
	if v, ok := os.LookupEnv("NM_WEBUI_TLS"); ok {
		cfg.TLS = v == "1" || strings.EqualFold(v, "true")
	}
	if v, ok := os.LookupEnv("NM_WEBUI_TLS_CERT"); ok {
		cfg.TLSCert = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_TLS_KEY"); ok {
		cfg.TLSKey = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_CONFIG"); ok {
		cfg.ConfigFile = v
	}
	if v, ok := os.LookupEnv("NM_WEBUI_PORTAL_URLS"); ok {
		cfg.CaptivePortalURLs = splitList(v)
	}
	if v, ok := os.LookupEnv("NM_WEBUI_PORTAL_ALLOW_JS"); ok {
		cfg.PortalAllowJS = v == "1" || strings.EqualFold(v, "true")
	}
	if v, ok := os.LookupEnv("NM_WEBUI_POWER_ACTION_PASSWORD"); ok {
		cfg.PowerActionPass = v
	}

	// 2. Parse CLI flags
	fs := flag.NewFlagSet("nm-webui", flag.ContinueOnError)
	var (
		flagListen          string
		flagAuthPass        string
		flagInterfaceFilter string
		flagTLS             bool
		flagTLSCert         string
		flagTLSKey          string
		flagLogLevel        string
		flagConfigFile      string
		flagConnectTimeout  int
		flagPortalURLs      string
		flagPortalAllowJS   bool
		flagPowerActionPass string
		flagVersion         bool
		flagVersionShort    bool
	)

	fs.StringVar(&flagListen, "listen", cfg.Listen, "listen address (IP:port)")
	fs.StringVar(&flagAuthPass, "auth-pass", cfg.AuthPass, "admin password for Basic Auth; empty = open access")
	fs.StringVar(&flagInterfaceFilter, "interface-filter", cfg.InterfaceFilter, "regex to filter visible interfaces (default is physical-only)")
	fs.BoolVar(&flagTLS, "tls", cfg.TLS, "enable HTTPS with an auto-generated or provided certificate")
	fs.StringVar(&flagTLSCert, "tls-cert", cfg.TLSCert, "path to TLS certificate (PEM)")
	fs.StringVar(&flagTLSKey, "tls-key", cfg.TLSKey, "path to TLS key (PEM)")
	fs.StringVar(&flagLogLevel, "log-level", cfg.LogLevel, "log level: debug, info, warn, error")
	fs.StringVar(&flagConfigFile, "config", cfg.ConfigFile, "path to config.yaml")
	fs.IntVar(&flagConnectTimeout, "connect-timeout", cfg.ConnectTimeout, "wifi connect timeout in seconds")
	fs.StringVar(&flagPortalURLs, "portal-check-urls", strings.Join(cfg.CaptivePortalURLs, ","), "comma-separated probe URLs for captive-portal detection")
	fs.BoolVar(&flagPortalAllowJS, "portal-allow-js", cfg.PortalAllowJS, "allow portal JavaScript to run (sandboxed iframe; disable to strip scripts)")
	fs.StringVar(&flagPowerActionPass, "power-action-password", cfg.PowerActionPass, "password for the Power section (reboot/poweroff); empty = Power section disabled")
	fs.BoolVar(&flagVersion, "version", false, "print version and exit")
	fs.BoolVar(&flagVersionShort, "v", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "nm-webui - NetworkManager web interface\n\nUsage:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, err
	}

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if flagVersion || flagVersionShort {
		cfg.ShowVersion = true
		return cfg, nil
	}

	// 3. Apply config file (overrides env and defaults, but not explicit CLI flags)
	if set["config"] {
		cfg.ConfigFile = flagConfigFile
	}
	if cfg.ConfigFile != "" {
		if err := cfg.applyConfigFile(set); err != nil {
			return nil, err
		}
	}

	// 4. Explicit CLI flags override everything
	if set["listen"] {
		cfg.Listen = flagListen
	}
	if set["auth-pass"] {
		cfg.AuthPass = flagAuthPass
	}
	if set["interface-filter"] {
		cfg.InterfaceFilter = flagInterfaceFilter
	}
	if set["tls"] {
		cfg.TLS = flagTLS
	}
	if set["tls-cert"] {
		cfg.TLSCert = flagTLSCert
	}
	if set["tls-key"] {
		cfg.TLSKey = flagTLSKey
	}
	if set["log-level"] {
		cfg.LogLevel = flagLogLevel
	}
	if set["connect-timeout"] {
		cfg.ConnectTimeout = flagConnectTimeout
	}
	if set["portal-check-urls"] {
		cfg.CaptivePortalURLs = splitList(flagPortalURLs)
	}
	if set["portal-allow-js"] {
		cfg.PortalAllowJS = flagPortalAllowJS
	}
	if set["power-action-password"] {
		cfg.PowerActionPass = flagPowerActionPass
	}

	return cfg.finalize(), nil
}

func (c *Config) applyConfigFile(set map[string]bool) error {
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	type fileCfg struct {
		Listen          *string `yaml:"listen"`
		AuthPass        *string `yaml:"auth-pass"`
		InterfaceFilter *string `yaml:"interface-filter"`
		TLS             *bool   `yaml:"tls"`
		TLSCert         *string `yaml:"tls-cert"`
		TLSKey          *string `yaml:"tls-key"`
		LogLevel        *string `yaml:"log-level"`
		ConnectTimeout  *int    `yaml:"connect-timeout"`
		PortalURLs      *string `yaml:"portal-check-urls"`
		PortalAllowJS   *bool   `yaml:"portal-allow-js"`
		PowerActionPass *string `yaml:"power-action-password"`
	}
	var fc fileCfg
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	if !set["listen"] && fc.Listen != nil {
		c.Listen = *fc.Listen
	}
	if !set["auth-pass"] && fc.AuthPass != nil {
		c.AuthPass = *fc.AuthPass
	}
	if !set["interface-filter"] && fc.InterfaceFilter != nil {
		c.InterfaceFilter = *fc.InterfaceFilter
	}
	if !set["tls"] && fc.TLS != nil {
		c.TLS = *fc.TLS
	}
	if !set["tls-cert"] && fc.TLSCert != nil {
		c.TLSCert = *fc.TLSCert
	}
	if !set["tls-key"] && fc.TLSKey != nil {
		c.TLSKey = *fc.TLSKey
	}
	if !set["log-level"] && fc.LogLevel != nil {
		c.LogLevel = *fc.LogLevel
	}
	if !set["connect-timeout"] && fc.ConnectTimeout != nil {
		c.ConnectTimeout = *fc.ConnectTimeout
	}
	if !set["portal-check-urls"] && fc.PortalURLs != nil {
		c.CaptivePortalURLs = splitList(*fc.PortalURLs)
	}
	if !set["portal-allow-js"] && fc.PortalAllowJS != nil {
		c.PortalAllowJS = *fc.PortalAllowJS
	}
	if !set["power-action-password"] && fc.PowerActionPass != nil {
		c.PowerActionPass = *fc.PowerActionPass
	}
	return nil
}

// splitList splits a comma-separated configuration value into a trimmed list.
func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) finalize() *Config {
	if c.InterfaceFilter == "" {
		c.InterfaceFilter = DefaultInterfaceFilter
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 45
	}
	var err error
	c.InterfaceRe, err = regexp.Compile(c.InterfaceFilter)
	if err != nil {
		c.InterfaceRe = regexp.MustCompile(DefaultInterfaceFilter)
	}
	return c
}

// InterfaceAllowed reports whether an interface should be visible.
func (c *Config) InterfaceAllowed(iface string) bool {
	return c.InterfaceRe.MatchString(iface)
}

// AuthEnabled reports whether a password is configured.
func (c *Config) AuthEnabled() bool {
	return c.AuthPass != ""
}

// PowerActionEnabled reports whether the Power section (reboot/poweroff)
// is active. It is enabled when a power action password is configured.
func (c *Config) PowerActionEnabled() bool {
	return c.PowerActionPass != ""
}
