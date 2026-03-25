package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	middleware "github.com/religiosa1/doors_of_durin/internal/http/middleware"
	"github.com/religiosa1/doors_of_durin/internal/ratelimit"
	"github.com/religiosa1/doors_of_durin/internal/service"
)

type Verify struct {
	AuthService     service.AuthService
	EnableBasicAuth bool
	Limiter         *ratelimit.Limiter
}

func (v Verify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	// BasicAuth flow
	if username, password, hasBasicAuth := r.BasicAuth(); hasBasicAuth {
		ip := middleware.GetRequestInfo(r.Context()).IP
		logger = logger.With(slog.String("username", username), slog.String("ip", ip))
		if !v.EnableBasicAuth {
			logger.Warn("basic auth attempt, while basic auth disabled in the config")
			http.Error(w, "basic auth is disabled for this service", http.StatusForbidden)
			return
		}
		if v.Limiter.IsBlocked(ip) {
			logger.Warn("basic auth blocked for the IP, through the rate limiter")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		loginErr := v.AuthService.LoginNoSession(username, password)
		if loginErr == nil {
			logger.Info("basic auth login")
			w.Header().Set("X-Auth-User", username)
			w.WriteHeader(http.StatusOK)
		} else {
			v.Limiter.RecordFailure(ip)
			logger.Error("bad basic auth login attempt", slog.Any("error", loginErr))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
		return
	}

	// Session cookie flow
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	username, err := v.AuthService.CheckAuth(cookie.Value)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) || errors.Is(err, service.ErrSessionExpired) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		logger.Error("unexpected error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Auth-User", username)
	w.WriteHeader(http.StatusOK)
}
