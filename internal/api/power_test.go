package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ru-ace/nm-webui/internal/config"
)

// fakePower records Check/Reboot/PowerOff calls so the handlers can be tested
// without touching the host.
type fakePower struct {
	mu       sync.Mutex
	calls    []string
	checks   []string
	checkErr error
}

func (f *fakePower) Reboot() error {
	f.mu.Lock()
	f.calls = append(f.calls, "reboot")
	f.mu.Unlock()
	return nil
}

func (f *fakePower) PowerOff() error {
	f.mu.Lock()
	f.calls = append(f.calls, "poweroff")
	f.mu.Unlock()
	return nil
}

func (f *fakePower) Check(action string) error {
	f.mu.Lock()
	f.checks = append(f.checks, action)
	err := f.checkErr
	f.mu.Unlock()
	return err
}

func (f *fakePower) callCount(action string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == action {
			n++
		}
	}
	return n
}

func newPowerTestServer(t *testing.T, cfg *config.Config, p powerActions) *httptest.Server {
	t.Helper()
	s := &Server{cfg: cfg, power: p, powerGuard: newAttemptLimiter(time.Minute, 8)}
	r := chi.NewRouter()
	r.Get("/system/features", s.handleSystemFeatures)
	r.Post("/system/power", s.handlePowerAction)
	return httptest.NewServer(r)
}

func doPowerPost(t *testing.T, base, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(base+"/system/power", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("power post: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String()
}

func doFeaturesGet(t *testing.T, base string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(base + "/system/features")
	if err != nil {
		t.Fatalf("features get: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("features decode: %v", err)
	}
	return out
}

func TestSystemFeaturesPowerEnabled(t *testing.T) {
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, &fakePower{})
	defer srv.Close()

	feat := doFeaturesGet(t, srv.URL)
	if v, _ := feat["power"].(bool); !v {
		t.Fatalf("features = %v, want power=true", feat)
	}
}

func TestSystemFeaturesPowerDisabled(t *testing.T) {
	cfg := &config.Config{}
	srv := newPowerTestServer(t, cfg, &fakePower{})
	defer srv.Close()

	feat := doFeaturesGet(t, srv.URL)
	if v, _ := feat["power"].(bool); v {
		t.Fatalf("features = %v, want power=false", feat)
	}
}

func TestSystemFeaturesPortalProxyBase(t *testing.T) {
	// A helper that fetches features with a pinned Host header (unit servers
	// bind a random port, so the base derivation must not be tested through
	// their ephemeral addresses).
	getFeaturesWithHost := func(srv *httptest.Server, host string) map[string]interface{} {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/system/features", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	cfg := &config.Config{Listen: "0.0.0.0:8090", PortalProxyListen: "0.0.0.0:8091"}
	srv := newPowerTestServer(t, cfg, &fakePower{})
	defer srv.Close()

	// Direct access on the admin port → 8091.
	if base, _ := getFeaturesWithHost(srv, "192.168.1.5:8090")["portal_proxy_base"].(string); base != "http://192.168.1.5:8091" {
		t.Fatalf("direct: portal_proxy_base = %q, want http://192.168.1.5:8091", base)
	}
	// Port-mapped forward → the offset carries over to the portal port.
	if base, _ := getFeaturesWithHost(srv, "127.0.0.1:18090")["portal_proxy_base"].(string); base != "http://127.0.0.1:18091" {
		t.Fatalf("forwarded: portal_proxy_base = %q, want http://127.0.0.1:18091", base)
	}

	// Disabled listener → empty base (the SPA falls back to the admin origin).
	cfg2 := &config.Config{Listen: "0.0.0.0:8090"}
	srv2 := newPowerTestServer(t, cfg2, &fakePower{})
	defer srv2.Close()
	if base, _ := getFeaturesWithHost(srv2, "192.168.1.5:8090")["portal_proxy_base"].(string); base != "" {
		t.Fatalf("portal_proxy_base = %q, want empty when listener disabled", base)
	}
}

func TestPowerActionDisabledReturnsNotFound(t *testing.T) {
	srv := newPowerTestServer(t, &config.Config{}, &fakePower{})
	defer srv.Close()

	status, _ := doPowerPost(t, srv.URL, `{"action":"reboot","password":"x"}`)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestPowerActionWrongPassword(t *testing.T) {
	p := &fakePower{}
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, p)
	defer srv.Close()

	status, body := doPowerPost(t, srv.URL, `{"action":"reboot","password":"nope"}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", status, body)
	}
	time.Sleep(20 * time.Millisecond) // give any stray goroutine a chance
	if p.callCount("reboot") != 0 {
		t.Fatal("action executed despite wrong password")
	}
}

func TestPowerActionUnknownAction(t *testing.T) {
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, &fakePower{})
	defer srv.Close()

	status, body := doPowerPost(t, srv.URL, `{"action":"format","password":"secret"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", status, body)
	}
}

func TestPowerActionRebootSuccess(t *testing.T) {
	old := powerResponseGrace
	powerResponseGrace = 2 * time.Millisecond
	defer func() { powerResponseGrace = old }()

	p := &fakePower{}
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, p)
	defer srv.Close()

	status, body := doPowerPost(t, srv.URL, `{"action":"reboot","password":"secret"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", status, body)
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(body), &out); err != nil || out["status"] != "rebooting" {
		t.Fatalf("unexpected body: %q", body)
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) && p.callCount("reboot") == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if p.callCount("reboot") != 1 {
		t.Fatalf("reboot executed %d times, want exactly 1", p.callCount("reboot"))
	}
}

func TestPowerActionPoweroffSuccess(t *testing.T) {
	old := powerResponseGrace
	powerResponseGrace = 2 * time.Millisecond
	defer func() { powerResponseGrace = old }()

	p := &fakePower{}
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, p)
	defer srv.Close()

	status, body := doPowerPost(t, srv.URL, `{"action":"poweroff","password":"secret"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", status, body)
	}
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) && p.callCount("poweroff") == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if p.callCount("poweroff") != 1 {
		t.Fatalf("poweroff executed %d times, want exactly 1", p.callCount("poweroff"))
	}
}

func TestPowerActionCheckFailure(t *testing.T) {
	old := powerResponseGrace
	powerResponseGrace = 2 * time.Millisecond
	defer func() { powerResponseGrace = old }()

	p := &fakePower{checkErr: errors.New("sudo not allowed")}
	cfg := &config.Config{PowerActionPass: "secret"}
	srv := newPowerTestServer(t, cfg, p)
	defer srv.Close()

	status, body := doPowerPost(t, srv.URL, `{"action":"reboot","password":"secret"}`)
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body: %s)", status, body)
	}
	time.Sleep(20 * time.Millisecond)
	if p.callCount("reboot") != 0 {
		t.Fatal("action executed despite failed capability check")
	}
}

func TestPowerActionRateLimit(t *testing.T) {
	s := &Server{
		cfg:        &config.Config{PowerActionPass: "secret"},
		power:      &fakePower{},
		powerGuard: newAttemptLimiter(time.Minute, 8),
	}
	// 8 wrong attempts: all rejected with 401.
	for i := 0; i < 8; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/system/power", bytes.NewBufferString(`{"action":"reboot","password":"nope"}`))
		s.handlePowerAction(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	// The next attempt is rate limited.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/system/power", bytes.NewBufferString(`{"action":"reboot","password":"secret"}`))
	s.handlePowerAction(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after 8 failures: status = %d, want 429", rec.Code)
	}
	if !s.powerGuard.isBlocked() {
		t.Fatal("powerGuard should be blocking after threshold")
	}
}

func TestPowerActionCorrectPasswordClearsLimiter(t *testing.T) {
	s := &Server{
		cfg:        &config.Config{PowerActionPass: "secret"},
		power:      &fakePower{},
		powerGuard: newAttemptLimiter(time.Minute, 8),
	}
	// 7 failures, then one success clears the counter.
	for i := 0; i < 7; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/system/power", bytes.NewBufferString(`{"action":"reboot","password":"nope"}`))
		s.handlePowerAction(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/system/power", bytes.NewBufferString(`{"action":"reboot","password":"secret"}`))
	s.handlePowerAction(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("success: status = %d, want 200", rec.Code)
	}
	if s.powerGuard.isBlocked() {
		t.Fatal("powerGuard should not be blocked after a successful attempt")
	}
}
