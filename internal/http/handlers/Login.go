package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/csrf"
	middleware "github.com/religiosa1/doors_of_durin/internal/http/middleware"
	"github.com/religiosa1/doors_of_durin/internal/ratelimit"
	"github.com/religiosa1/doors_of_durin/internal/service"
	views "github.com/religiosa1/doors_of_durin/internal/views"
)

type loginData struct {
	StaticURL   string
	LoginAction string
	RedirectTo  string
	Error       string
	CSRFToken   string
}

// prefixedPath returns a relative path when prefix is empty, or an absolute
// prefixed path otherwise. Templates rely on this to work both standalone
// (no prefix → relative refs) and behind a reverse proxy (prefix → absolute).
func prefixedPath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	jp, err := url.JoinPath(prefix, path)
	// probably an overkill, but why not
	if err != nil {
		return path
	}
	return jp
}

type Login struct {
	URLPrefix string
}

func (l Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		logger.Error("failed to generate CSRF token", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	csrf.SetCookie(w, csrfToken)
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = middleware.GetRequestInfo(r.Context()).OriginalURL()
	}
	data := loginData{
		StaticURL:   prefixedPath(l.URLPrefix, "static"),
		LoginAction: prefixedPath(l.URLPrefix, "login"),
		RedirectTo:  redirectTo,
		CSRFToken:   csrfToken,
	}
	if err := views.Render(w, "login.gohtml", data); err != nil {
		logger.Error("failed to render login page", slog.Any("error", err))
	}
}

type LoginSubmit struct {
	AuthService service.AuthService
	Limiter     *ratelimit.Limiter
	SessionTTL  time.Duration
	URLPrefix   string
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
			StaticURL:   prefixedPath(l.URLPrefix, "static"),
			LoginAction: prefixedPath(l.URLPrefix, "login"),
			RedirectTo:  redirectTo,
			Error:       "Invalid username or password",
			CSRFToken:   newToken,
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
