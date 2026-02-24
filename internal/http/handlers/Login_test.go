package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/religiosa1/auth_server/internal/http/handlers"
	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
	"github.com/religiosa1/auth_server/internal/repository/users"
)

func newTestDB(t *testing.T) *repository.DB {
	t.Helper()
	db, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func loginRequest(t *testing.T, username, password, redirectTo string) *http.Request {
	t.Helper()
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	if redirectTo != "" {
		form.Set("redirect_to", redirectTo)
	}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func TestLoginSubmit_Success_NoRedirectTo(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{DB: db}.ServeHTTP(rr, loginRequest(t, "alice", "secret", ""))

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
	if err := sessions.CheckSessionExists(*db, cookie.Value); err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
}

func TestLoginSubmit_Success_WithRedirectTo(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "secret"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{DB: db}.ServeHTTP(rr, loginRequest(t, "alice", "secret", "/dashboard"))

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
	if err := sessions.CheckSessionExists(*db, cookie.Value); err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
}

func TestLoginSubmit_WrongPassword(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "correct"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.LoginSubmit{DB: db}.ServeHTTP(rr, loginRequest(t, "alice", "wrong", ""))

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
	handlers.LoginSubmit{DB: db}.ServeHTTP(rr, loginRequest(t, "nobody", "password", ""))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if cookie := sessionCookie(rr); cookie != nil {
		t.Fatal("expected no session_id cookie for unknown user")
	}
}
