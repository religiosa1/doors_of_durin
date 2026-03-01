package middleware_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/religiosa1/doors_of_durin/internal/http/middleware"
)

// captureRequestInfo runs a request through the WithLogger middleware and
// returns the RequestInfo stored in the context.
func captureRequestInfo(t *testing.T, req *http.Request) middleware.RequestInfo {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var captured middleware.RequestInfo
	handler := middleware.WithLogger(logger)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = middleware.GetRequestInfo(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return captured
}

func TestWithLogger_RequestInfo_NoForwardingHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	req.Host = "auth.example.com"
	info := captureRequestInfo(t, req)

	if !info.SchemeFallback {
		t.Error("SchemeFallback should be true without X-Forwarded-Proto")
	}
	if info.Scheme == "" {
		t.Error("Scheme should have a fallback value")
	}
	if !info.HostFallback {
		t.Error("HostFallback should be true without X-Forwarded-Host")
	}
	if info.Host != "auth.example.com" {
		t.Errorf("Host = %q, want %q", info.Host, "auth.example.com")
	}
	if info.URI != "" {
		t.Errorf("URI = %q, want empty", info.URI)
	}
	if info.Path != "/login" {
		t.Errorf("Path = %q, want %q", info.Path, "/login")
	}
}

func TestWithLogger_RequestInfo_XForwardedProto(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	info := captureRequestInfo(t, req)

	if info.SchemeFallback {
		t.Error("SchemeFallback should be false with X-Forwarded-Proto")
	}
	if info.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", info.Scheme, "https")
	}
}

func TestWithLogger_RequestInfo_XForwardedHost(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	info := captureRequestInfo(t, req)

	if info.HostFallback {
		t.Error("HostFallback should be false with X-Forwarded-Host")
	}
	if info.Host != "app.example.com" {
		t.Errorf("Host = %q, want %q", info.Host, "app.example.com")
	}
}

func TestWithLogger_RequestInfo_XForwardedURI(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	req.Header.Set("X-Forwarded-URI", "/protected?q=1")
	info := captureRequestInfo(t, req)

	if info.URI != "/protected?q=1" {
		t.Errorf("URI = %q, want %q", info.URI, "/protected?q=1")
	}
	if info.Path != "/login" {
		t.Errorf("Path = %q, want %q (current path, not forwarded URI)", info.Path, "/login")
	}
}

func TestWithLogger_RequestInfo_AllForwardingHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-URI", "/protected")
	info := captureRequestInfo(t, req)

	want := "https://app.example.com/protected"
	if got := info.OriginalURL(); got != want {
		t.Errorf("OriginalURL() = %q, want %q", got, want)
	}
}
