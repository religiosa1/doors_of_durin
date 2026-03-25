package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/csrf"
	"github.com/religiosa1/doors_of_durin/internal/http/handlers"
	"github.com/religiosa1/doors_of_durin/internal/ratelimit"
	"github.com/religiosa1/doors_of_durin/internal/repository"
	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
	"github.com/religiosa1/doors_of_durin/internal/repository/users"
	"github.com/religiosa1/doors_of_durin/internal/service"
)

func newTestDB(t *testing.T) *repository.DB {
	t.Helper()
	db, err := repository.New(":memory:", nil)
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func loginRequest(t *testing.T, username, password, redirectTo string) *http.Request {
	t.Helper()
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		t.Fatalf("loginRequest: generate CSRF token: %v", err)
	}
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set(csrf.FormFieldName, csrfToken)
	if redirectTo != "" {
		form.Set("redirect_to", redirectTo)
	}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: csrfToken})
	return req
}

func sessionCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == handlers.SessionCookieName {
			return c
		}
	}
	return nil
}

func TestLoginSubmit_CSRF_MissingCookie(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	csrfToken, _ := csrf.GenerateToken()
	form := url.Values{}
	form.Set("username", "alice")
	form.Set("password", "secret")
	form.Set(csrf.FormFieldName, csrfToken)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// no CSRF cookie added

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestLoginSubmit_CSRF_MissingFormField(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	csrfToken, _ := csrf.GenerateToken()
	form := url.Values{}
	form.Set("username", "alice")
	form.Set("password", "secret")
	// no csrf_token form field
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: csrfToken})

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestLoginSubmit_CSRF_TokenMismatch(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	cookieToken, _ := csrf.GenerateToken()
	formToken, _ := csrf.GenerateToken()
	form := url.Values{}
	form.Set("username", "alice")
	form.Set("password", "secret")
	form.Set(csrf.FormFieldName, formToken)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: cookieToken})

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestLoginSubmit_Success_NoRedirectTo(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, loginRequest(t, "alice", "secret", ""))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}

	cookie := sessionCookie(rr)
	if cookie == nil {
		t.Fatal("expected session_id cookie to be set")
	}
	if _, err := sessions.GetSession(*db, cookie.Value); err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
}

func TestLoginSubmit_Success_WithRedirectTo(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, loginRequest(t, "alice", "secret", "/dashboard"))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", loc)
	}

	cookie := sessionCookie(rr)
	if cookie == nil {
		t.Fatal("expected session_id cookie to be set")
	}
	if _, err := sessions.GetSession(*db, cookie.Value); err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
}

func TestLoginSubmit_WrongPassword(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "correct"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, loginRequest(t, "alice", "wrong", ""))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if cookie := sessionCookie(rr); cookie != nil {
		t.Fatal("expected no session_id cookie on failed login")
	}
}

func TestLoginSubmit_UnknownUser(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, loginRequest(t, "nobody", "password", ""))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if cookie := sessionCookie(rr); cookie != nil {
		t.Fatal("expected no session_id cookie for unknown user")
	}
}

func TestLoginSubmit_RateLimit_BlocksAfterThreshold(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: 3, Window: time.Minute, FailDelay: 0})
	h := handlers.LoginSubmit{AuthService: service.AuthService{DB: db}, Limiter: limiter}

	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, loginRequest(t, "alice", "wrong", ""))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected %d, got %d", i+1, http.StatusUnauthorized, rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, loginRequest(t, "alice", "secret", ""))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d after threshold, got %d", http.StatusTooManyRequests, rr.Code)
	}
}

func TestLoginSubmit_RateLimit_SuccessDoesNotIncrement(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: 1, Window: time.Minute, FailDelay: 0})
	h := handlers.LoginSubmit{AuthService: service.AuthService{DB: db}, Limiter: limiter}

	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, loginRequest(t, "alice", "secret", ""))
		if rr.Code != http.StatusSeeOther {
			t.Fatalf("successful login %d: expected %d, got %d", i+1, http.StatusSeeOther, rr.Code)
		}
	}
}

func TestLoginSubmit_FailDelay(t *testing.T) {
	db := newTestDB(t)

	const delay = 20 * time.Millisecond
	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: 100, Window: time.Minute, FailDelay: delay})
	h := handlers.LoginSubmit{AuthService: service.AuthService{DB: db}, Limiter: limiter}

	start := time.Now()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, loginRequest(t, "nobody", "password", ""))
	elapsed := time.Since(start)

	if elapsed < delay {
		t.Fatalf("expected fail delay >= %v, got %v", delay, elapsed)
	}
}
