package handlers

import (
	"log/slog"
	"net/http"

	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	views "github.com/religiosa1/auth_server/internal/views"
)

type loginData struct {
	RedirectTo string
	Error      string
}

type Login struct{}

func (l Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := loginData{
		RedirectTo: r.URL.Query().Get("redirect_to"),
	}
	if err := views.Render(w, "login.gohtml", data); err != nil {
		logger := middleware.GetLogger(r.Context())
		logger.Error("failed to render login page", slog.Any("error", err))
	}
}

type LoginSubmit struct{}

func (l LoginSubmit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	redirectTo := r.FormValue("redirect_to")

	_ = username
	// TODO: look up user, verify password, create session, set cookie

	logger.Info("login attempt", slog.String("username", username))

	data := loginData{
		RedirectTo: redirectTo,
		Error:      "Invalid username or password",
	}
	w.WriteHeader(http.StatusUnauthorized)
	if err := views.Render(w, "login.gohtml", data); err != nil {
		logger.Error("failed to render login page", slog.Any("error", err))
	}
}
