package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ru-ace/nm-webui/internal/config"
	"github.com/ru-ace/nm-webui/internal/events"
	"github.com/ru-ace/nm-webui/internal/nm"
	"github.com/ru-ace/nm-webui/internal/system"
)

// Server holds the HTTP handlers and their dependencies.
type Server struct {
	cfg        *config.Config
	nm         *nm.Client
	hub        *events.Hub
	resolver   *system.Resolver
	portal     *system.Detector
	power      powerActions
	powerGuard *attemptLimiter
	timeout    time.Duration
}

// New creates the API server.
func New(cfg *config.Config, client *nm.Client, hub *events.Hub) *Server {
	jar, _ := cookiejar.New(nil)
	return &Server{
		cfg:        cfg,
		nm:         client,
		hub:        hub,
		resolver:   system.NewResolver(),
		portal:     system.NewDetector(cfg.CaptivePortalURLs, 30*time.Second, portalProbeWait, jar, cfg.PortalAllowJS),
		power:      system.NewPowerController(),
		powerGuard: newAttemptLimiter(time.Minute, 8),
		timeout:    time.Duration(cfg.ConnectTimeout) * time.Second,
	}
}

// Handler builds the full HTTP handler tree.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(middleware.Compress(5))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Group(func(r chi.Router) {
		r.Use(BasicAuth(s.cfg))
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/system/status", s.handleSystemStatus)
			r.Get("/system/features", s.handleSystemFeatures)
			r.Post("/system/external-ip/refresh", s.handleExternalIPRefresh)
			r.Post("/system/power", s.handlePowerAction)
			r.Get("/system/captive-portal", s.handleCaptivePortalStatus)
			r.Post("/system/captive-portal/check", s.handleCaptivePortalCheck)
			r.Get("/captive-portal/proxy", s.handlePortalProxyGet)
			r.Post("/captive-portal/proxy", s.handlePortalProxyPost)
			r.Get("/devices", s.handleDeviceList)
			r.Get("/devices/{iface}", s.handleDeviceGet)
			r.Post("/devices/{iface}/disconnect", s.handleDeviceDisconnect)
			r.Post("/devices/{iface}/up", s.handleDeviceUp)
			r.Route("/wifi", func(r chi.Router) {
				r.Post("/{iface}/scan", s.handleWifiScan)
				r.Get("/{iface}/networks", s.handleWifiNetworks)
				r.Post("/{iface}/connect", s.handleWifiConnect)
				r.Get("/{iface}/status", s.handleWifiStatus)
			})
			r.Route("/connections", func(r chi.Router) {
				r.Get("/", s.handleConnectionsList)
				r.Post("/", s.handleConnectionsCreate)
				r.Delete("/{uuid}", s.handleConnectionsDelete)
				r.Put("/{uuid}", s.handleConnectionsUpdate)
				r.Post("/{uuid}/up", s.handleConnectionsUp)
				r.Post("/{uuid}/down", s.handleConnectionsDown)
			})
			r.Get("/events", s.handleEvents)
		})
	})

	r.Mount("/", s.spaHandler())
	return r
}

// StartBridge launches the D-Bus → SSE transposition loop.
func (s *Server) StartBridge(ctx context.Context) error {
	slog.Info("starting D-Bus event bridge")
	return s.startBridge(ctx)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		if r.URL.Path != "/api/v1/events" {
			slog.Info("http", "method", r.Method, "path", r.URL.Path,
				"status", ww.Status(), "dur", time.Since(start).Round(time.Microsecond).String())
		}
	})
}
