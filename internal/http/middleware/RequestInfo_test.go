package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/religiosa1/doors_of_durin/internal/http/middleware"
)

func TestXForwardForStripping(t *testing.T) {
	const remoteAddr = "192.51.100.1"
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			"uses a single value from X-Forwarded-For",
			"198.51.100.2",
			"198.51.100.2",
		},
		{
			"extracts the first value from X-Forwarded-For, if multiple are provided",
			"198.51.100.3, 198.51.100.4",
			"198.51.100.3",
		},
		{
			"falls back to remoteAddr if no header is provided",
			"",
			remoteAddr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = remoteAddr
			r.Header.Set("X-Forwarded-For", tt.header)
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
