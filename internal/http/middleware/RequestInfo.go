package middleware

import "net/http"

// RequestInfo is basic information about the incoming http request
type RequestInfo struct {
	// Request Scheme -- http or https
	Scheme string
	// Request method
	Method string
	// Request host
	Host string
	// Request path
	Path string
	// Remote IP
	IP string
}

// ParseInfo parses Info from the request
func parseRequestInfo(r *http.Request) RequestInfo {
	var info RequestInfo

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		info.Scheme = proto
	} else if r.TLS != nil {
		info.Scheme = "https"
	} else {
		info.Scheme = "http"
	}
	if method := r.Header.Get("X-Original-Method"); method != "" {
		info.Method = method
	} else {
		info.Method = r.Method
	}
	if path := r.Header.Get("X-Forwarded-URI"); path != "" {
		info.Path = path
	} else {
		info.Path = r.URL.Path
	}
	if remoteAddr := r.Header.Get("X-Forwarded-For"); remoteAddr != "" {
		info.IP = remoteAddr
	} else {
		info.IP = r.RemoteAddr
	}

	return info
}
