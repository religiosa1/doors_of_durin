package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
)

type Logout struct {
	DB *repository.DB
}

func (l Logout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := sessions.DeleteSession(*l.DB, cookie.Value); err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
		logger.Error("error deleting session", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	target := r.URL.Query().Get("back_url")
	if target == "" {
		target = "/"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
