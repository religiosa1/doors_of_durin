package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/religiosa1/doors_of_durin/internal/http/handlers"
	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
	"github.com/religiosa1/doors_of_durin/internal/service"
)

func logoutRequest(sessionID string, backURL string) *http.Request {
	target := "/logout"
	if backURL != "" {
		target += "?" + url.Values{"back_url": {backURL}}.Encode()
	}
	req := httptest.NewRequest("POST", target, nil)
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: handlers.SessionCookieName, Value: sessionID})
	}
	return req
}

func TestLogout_NoCookie(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest("", ""))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestLogout_ValidSession_RedirectsToRoot(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest(sessionID, ""))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
}

func TestLogout_ValidSession_DeletesSessionFromDB(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest(sessionID, ""))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if _, err := sessions.GetSession(*db, sessionID); err == nil {
		t.Fatal("expected session to be deleted from DB, but it still exists")
	}
}

func TestLogout_ValidSession_ClearsCookie(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest(sessionID, ""))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rr.Code)
	}
	cookie := sessionCookie(rr)
	if cookie == nil {
		t.Fatal("expected Set-Cookie header for session_id")
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("expected MaxAge < 0 to expire cookie, got %d", cookie.MaxAge)
	}
}

func TestLogout_WithBackURL(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest(sessionID, "/goodbye"))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/goodbye" {
		t.Fatalf("expected redirect to /goodbye, got %q", loc)
	}
}

func TestLogout_StaleSessionCookie_SucceedsAndClearsCookie(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.Logout{AuthService: service.AuthService{DB: db}}.ServeHTTP(rr, logoutRequest("01JMVNP7HZE8Q3K5X2WRT4Y6BD", ""))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rr.Code)
	}
	cookie := sessionCookie(rr)
	if cookie == nil {
		t.Fatal("expected Set-Cookie header for session_id even for stale session")
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("expected MaxAge < 0 to expire cookie, got %d", cookie.MaxAge)
	}
}
