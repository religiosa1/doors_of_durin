package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
)

type Verify struct {
	DB         *repository.DB
	SessionTTL time.Duration
}

func (v Verify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var notBefore time.Time
	if v.SessionTTL > 0 {
		notBefore = time.Now().Add(-v.SessionTTL)
	}

	session, err := sessions.GetSession(*v.DB, cookie.Value, notBefore)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		logger.Error("error checking session", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := sessions.RegisterSessionUsage(*v.DB, cookie.Value); err != nil {
		logger.Error("error registering session usage", slog.Any("error", err))
	}

	w.Header().Set("X-Auth-User", session.Username)
	w.WriteHeader(http.StatusOK)
}
