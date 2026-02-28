package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/religiosa1/auth_server/internal/csrf"
	middleware "github.com/religiosa1/auth_server/internal/http/middleware"
	"github.com/religiosa1/auth_server/internal/ratelimit"
	"github.com/religiosa1/auth_server/internal/service"
	views "github.com/religiosa1/auth_server/internal/views"
)

type loginData struct {
	RedirectTo string
	Error      string
	CSRFToken  string
}

type Login struct{}

func (l Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		logger.Error("failed to generate CSRF token", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	csrf.SetCookie(w, csrfToken)
	data := loginData{
		RedirectTo: middleware.GetRequestInfo(r.Context()).OriginalURL(),
		CSRFToken:  csrfToken,
	}
	if err := views.Render(w, "login.gohtml", data); err != nil {
		logger.Error("failed to render login page", slog.Any("error", err))
	}
}

type LoginSubmit struct {
	AuthService service.AuthService
	Limiter     *ratelimit.Limiter
	SessionTTL  time.Duration
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

	if !csrf.ValidateToken(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	redirectTo := r.FormValue("redirect_to")

	logger.Info("login attempt", slog.String("username", username))

	renderUnauthorized := func() {
		newToken, err := csrf.GenerateToken()
		if err != nil {
			logger.Error("failed to generate CSRF token", slog.Any("error", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		csrf.SetCookie(w, newToken)
		data := loginData{
			RedirectTo: redirectTo,
			Error:      "Invalid username or password",
			CSRFToken:  newToken,
		}
		w.WriteHeader(http.StatusUnauthorized)
		if err := views.Render(w, "login.gohtml", data); err != nil {
			logger.Error("failed to render login page", slog.Any("error", err))
		}
	}

	sessionID, err := l.AuthService.Login(username, password)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrWrongPassword) {
			l.recordFailure(r)
			renderUnauthorized()
			return
		}
		logger.Error("error checking password", slog.String("username", username), slog.Any("error", err))
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
