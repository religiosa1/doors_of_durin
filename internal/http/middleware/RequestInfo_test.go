package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/religiosa1/doors_of_durin/internal/http/middleware"
)

func TestIPResolution(t *testing.T) {
	const remoteAddr = "192.51.100.1"
	tests := []struct {
		name    string
		realIP  string
		xff     string
		want    string
	}{
		{
			name:   "uses X-Real-IP when set",
			realIP: "198.51.100.2",
			want:   "198.51.100.2",
		},
		{
			name:   "X-Real-IP takes precedence over X-Forwarded-For",
			realIP: "198.51.100.2",
			xff:    "198.51.100.9",
			want:   "198.51.100.2",
		},
		{
			name: "falls back to first X-Forwarded-For entry when X-Real-IP absent",
			xff:  "198.51.100.3",
			want: "198.51.100.3",
		},
		{
			name: "extracts first value from X-Forwarded-For when multiple are provided",
			xff:  "198.51.100.3, 198.51.100.4",
			want: "198.51.100.3",
		},
		{
			name: "falls back to remoteAddr when no headers are provided",
			want: remoteAddr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = remoteAddr
			if tt.realIP != "" {
				r.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			info := middleware.ParseRequestInfo(r)
			if got := info.IP; tt.want != got {
				t.Errorf("Unexpected IP addr value; want %q got %q", tt.want, got)
			}
		})
	}
}

func TestOriginalURL(t *testing.T) {
	tests := []struct {
		name string
		info middleware.RequestInfo
		want string
	}{
		{
			name: "no URI returns empty string",
			info: middleware.RequestInfo{
				Scheme: "https",
				Host:   "app.example.com",
			},
			want: "",
		},
		{
			name: "URI set, neither forwarded, returns relative",
			info: middleware.RequestInfo{
				Scheme: "https", SchemeFallback: true,
				Host: "app.example.com", HostFallback: true,
				URI: "/protected",
			},
			want: "/protected",
		},
		{
			name: "URI set, only scheme forwarded, returns relative",
			info: middleware.RequestInfo{
				Scheme: "https",
				Host:   "app.example.com", HostFallback: true,
				URI: "/protected",
			},
			want: "/protected",
		},
		{
			name: "URI set, only host forwarded, returns relative",
			info: middleware.RequestInfo{
				Scheme: "https", SchemeFallback: true,
				Host: "app.example.com",
				URI:  "/protected",
			},
			want: "/protected",
		},
		{
			name: "URI set, both forwarded, returns absolute URL",
			info: middleware.RequestInfo{
				Scheme: "https",
				Host:   "app.example.com",
				URI:    "/protected",
			},
			want: "https://app.example.com/protected",
		},
		{
			name: "URI with query string preserved",
			info: middleware.RequestInfo{
				Scheme: "https",
				Host:   "app.example.com",
				URI:    "/protected?next=1&tab=overview",
			},
			want: "https://app.example.com/protected?next=1&tab=overview",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.OriginalURL(); got != tt.want {
				t.Fatalf("OriginalURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
