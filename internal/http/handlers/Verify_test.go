package handlers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/http/handlers"
	"github.com/religiosa1/doors_of_durin/internal/repository"
	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
	"github.com/religiosa1/doors_of_durin/internal/repository/users"
	"github.com/religiosa1/doors_of_durin/internal/service"
)

func verifyRequest(sessionID string) *http.Request {
	req := httptest.NewRequest("GET", "/verify", nil)
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: handlers.SessionCookieName, Value: sessionID})
	}
	return req
}

func createTestSession(t *testing.T, db *repository.DB, username string) string {
	t.Helper()
	if err := users.Create(*db, username, "password"); err != nil {
		t.Fatalf("createTestSession users.Create: %v", err)
	}
	userID, err := users.GetUserID(*db, username)
	if err != nil {
		t.Fatalf("createTestSession users.GetUserID: %v", err)
	}
	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("createTestSession sessions.CreateSession: %v", err)
	}
	return sessionID
}

// backdateSession sets the session's created_at to now-age and clears last_used_at,
// simulating a session that was created in the past and never touched since.
func backdateSession(t *testing.T, db *repository.DB, sessionID string, age time.Duration) {
	t.Helper()
	pastTime := time.Now().Add(-age).UTC().Format("2006-01-02 15:04:05")
	_, err := db.DB.Exec(
		"UPDATE sessions SET created_at = ?, last_used_at = NULL WHERE id = ?",
		pastTime, sessionID,
	)
	if err != nil {
		t.Fatalf("backdateSession: %v", err)
	}
}

func TestVerify_ValidSession(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest(sessionID))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("X-Auth-User"); got != "alice" {
		t.Fatalf("expected X-Auth-User=alice, got %q", got)
	}
}

func TestVerify_NoCookie(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest(""))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestVerify_SessionNotInDB(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest("01JMVNP7HZE8Q3K5X2WRT4Y6BD"))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestVerify_MalformedCookieValue(t *testing.T) {
	db := newTestDB(t)

	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest("not-a-valid-session-id"))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestVerify_ExpiredSession(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")
	backdateSession(t, db, sessionID, 2*time.Hour)

	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest(sessionID))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestVerify_LastUsedAtBumped(t *testing.T) {
	db := newTestDB(t)
	sessionID := createTestSession(t, db, "alice")

	before := time.Now()
	rr := httptest.NewRecorder()
	handlers.Verify{AuthService: service.AuthService{DB: db, SessionTTL: time.Hour}}.ServeHTTP(rr, verifyRequest(sessionID))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var lastUsedAt sql.NullTime
	err := db.DB.QueryRow("SELECT last_used_at FROM sessions WHERE id = ?", sessionID).Scan(&lastUsedAt)
	if err != nil {
		t.Fatalf("querying last_used_at: %v", err)
	}
	if !lastUsedAt.Valid {
		t.Fatal("expected last_used_at to be set after Verify")
	}
	if lastUsedAt.Time.Before(before.Truncate(time.Second)) {
		t.Fatalf("expected last_used_at >= %v, got %v", before, lastUsedAt.Time)
	}
}
