package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	"github.com/religiosa1/auth_server/internal/service"
)

type Verify struct {
	AuthService service.AuthService
}

func (v Verify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

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
