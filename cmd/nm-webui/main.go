package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ru-ace/nm-webui/internal/api"
	"github.com/ru-ace/nm-webui/internal/config"
	"github.com/ru-ace/nm-webui/internal/events"
	"github.com/ru-ace/nm-webui/internal/nm"
	"github.com/ru-ace/nm-webui/internal/tlsutil"
)

var version = "dev"

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(2)
	}

	if cfg.ShowVersion {
		fmt.Println("nm-webui", version)
		os.Exit(0)
	}

	setupLogger(cfg.LogLevel)
	slog.Info("starting nm-webui", "version", version)

	client, err := nm.Connect()
	if err != nil {
		slog.Error("cannot connect to NetworkManager over D-Bus", "err", err)
		slog.Error("is NetworkManager running? check: systemctl status NetworkManager")
		os.Exit(1)
	}
	defer client.Close()

	hub := events.NewHub(50)
	server := api.New(cfg, client, hub)
	if cfg.PowerActionEnabled() {
		slog.Warn("power actions enabled: reboot/poweroff of the host are exposed through the web UI")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.StartBridge(ctx); err != nil {
		slog.Error("event bridge", "err", err)
		os.Exit(1)
	}
	// Portal checks run fully self-owned: NM verdicts are only a link-level
	// fallback and a trigger source, the probe is the judge.
	server.StartPortalMonitor(ctx)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	tlsConfig, err := tlsSetup(cfg)
	if err != nil {
		slog.Error("tls setup", "err", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("http server listening", "addr", cfg.Listen,
			"auth", cfg.AuthEnabled(), "tls", cfg.TLS)
		var err error
		if cfg.TLS {
			srv.TLSConfig = tlsConfig
			err = srv.ListenAndServeTLS("", "")
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func tlsSetup(cfg *config.Config) (*tls.Config, error) {
	if !cfg.TLS {
		return nil, nil
	}
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		return tlsutil.ConfigFromMaterial(&tlsutil.SelfSignedMaterial{CertFile: cfg.TLSCert, KeyFile: cfg.TLSKey})
	}
	certDir := "/var/lib/nm-webui"
	if err := os.MkdirAll(certDir, 0o750); err != nil {
		if home, hErr := os.UserHomeDir(); hErr == nil {
			certDir = filepath.Join(home, ".local", "share", "nm-webui")
		} else {
			certDir = "./certs"
		}
	}
	slog.Info("generating self-signed TLS certificate", "dir", certDir)
	m, err := tlsutil.GenerateSelfSigned(certDir)
	if err != nil {
		return nil, err
	}
	return tlsutil.ConfigFromMaterial(m)
}

func setupLogger(level string) {
	l := slog.LevelInfo
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
}
