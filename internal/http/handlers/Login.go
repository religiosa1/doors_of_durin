package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	"github.com/religiosa1/auth_server/internal/ratelimit"
	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
	"github.com/religiosa1/auth_server/internal/repository/users"
	views "github.com/religiosa1/auth_server/internal/views"
)

const SessionCookieName = "session_id"

type loginData struct {
	RedirectTo string
	Error      string
}

type Login struct{}

func (l Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := loginData{
		RedirectTo: middleware.GetRequestInfo(r.Context()).OriginalURL(),
	}
	if err := views.Render(w, "login.gohtml", data); err != nil {
		logger := middleware.GetLogger(r.Context())
		logger.Error("failed to render login page", slog.Any("error", err))
	}
}

type LoginSubmit struct {
	DB         *repository.DB
	Limiter    *ratelimit.Limiter
	SessionTTL time.Duration
}

func (l LoginSubmit) recordFailure(r *http.Request) {
	if l.Limiter == nil {
		return
	}
	ip := middleware.GetRequestInfo(r.Context()).IP
	l.Limiter.RecordFailure(ip)
	time.Sleep(l.Limiter.FailDelay())
}

func (l LoginSubmit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	if l.Limiter != nil {
		ip := middleware.GetRequestInfo(r.Context()).IP
		if l.Limiter.IsBlocked(ip) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	redirectTo := r.FormValue("redirect_to")

	logger.Info("login attempt", slog.String("username", username))

	renderUnauthorized := func() {
		data := loginData{
			RedirectTo: redirectTo,
			Error:      "Invalid username or password",
		}
		w.WriteHeader(http.StatusUnauthorized)
		if err := views.Render(w, "login.gohtml", data); err != nil {
			logger.Error("failed to render login page", slog.Any("error", err))
		}
	}

	ok, err := users.CheckPassword(*l.DB, username, password)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			l.recordFailure(r)
			renderUnauthorized()
			return
		}
		logger.Error("error checking password", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !ok {
		l.recordFailure(r)
		renderUnauthorized()
		return
	}

	userID, err := users.GetUserID(*l.DB, username)
	if err != nil {
		logger.Error("error getting user ID after successful auth", slog.String("username", username), slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sessionID, err := sessions.CreateSession(*l.DB, userID)
	if err != nil {
		logger.Error("error creating session", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// MaxAge 0 omits the attribute entirely, making this a session cookie — matches TTL=0 meaning no expiry on the server side too.
		MaxAge: int(l.SessionTTL.Seconds()),
	})

	target := redirectTo
	if target == "" {
		target = "/"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
