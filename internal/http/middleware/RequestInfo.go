package middleware

import (
	"net/http"
	"strings"
)

// RequestInfo is basic information about the incoming http request
type RequestInfo struct {
	// Original Request Scheme -- http or https; falls back to current scheme, if not forwarded via headers
	Scheme string
	// SchemeFallback is true if Scheme was inferred from the local connection
	// (unreliable behind a proxy) rather than captured from X-Forwarded-Proto.
	SchemeFallback bool
	// Original Request method; falls back to current method if not forwarded via headers
	Method string
	// Original Request host; falls back to current host if not forwarded via headers
	Host string
	// HostFallback is true if Host fell back to the Host header of the proxied
	// request (the auth server's own host) rather than being captured from X-Forwarded-Host.
	HostFallback bool
	// Original request URI
	URI string
	// Current (after proxying) Request path
	Path string
	// Remote IP
	IP string
}

// ParseRequestInfo parses Info from the request
func ParseRequestInfo(r *http.Request) RequestInfo {
	var info RequestInfo

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		info.Scheme = proto
	} else if r.TLS != nil {
		info.Scheme = "https"
		info.SchemeFallback = true
	} else {
		info.Scheme = "http"
		info.SchemeFallback = true
	}
	if method := r.Header.Get("X-Original-Method"); method != "" {
		info.Method = method
	} else {
		info.Method = r.Method
	}
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		info.Host = host
	} else {
		info.Host = r.Host
		info.HostFallback = true
	}
	info.URI = r.Header.Get("X-Forwarded-URI")
	info.Path = r.URL.Path
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		info.IP = xri
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		firstProxy, _, _ := strings.Cut(xff, ",")
		info.IP = strings.TrimSpace(firstProxy)
	} else {
		info.IP = r.RemoteAddr
	}

	return info
}

// OriginalURL reconstructs the full URL of the original request before the
// reverse proxy. Returns an absolute URL when both scheme and host were
// forwarded, a relative URL (URI only) when either was not, and an empty
// string if no forwarding headers were present at all.
func (info RequestInfo) OriginalURL() string {
	if info.URI == "" {
		return ""
	}
	if info.SchemeFallback || info.HostFallback {
		return info.URI
	}
	return info.Scheme + "://" + info.Host + info.URI
}
