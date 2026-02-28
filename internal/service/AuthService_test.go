package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
	"github.com/religiosa1/auth_server/internal/repository/users"
	"github.com/religiosa1/auth_server/internal/service"
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

func createTestUser(t *testing.T, db *repository.DB, username, password string) {
	t.Helper()
	if err := users.Create(*db, username, password); err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
}

func createTestSession(t *testing.T, db *repository.DB, username string) string {
	t.Helper()
	userID, err := users.GetUserID(*db, username)
	if err != nil {
		t.Fatalf("createTestSession GetUserID: %v", err)
	}
	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("createTestSession CreateSession: %v", err)
	}
	return sessionID
}

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

func setSessionLastUsed(t *testing.T, db *repository.DB, sessionID string, age time.Duration) {
	t.Helper()
	pastTime := time.Now().Add(-age).UTC().Format("2006-01-02 15:04:05")
	_, err := db.DB.Exec(
		"UPDATE sessions SET last_used_at = ? WHERE id = ?",
		pastTime, sessionID,
	)
	if err != nil {
		t.Fatalf("setSessionLastUsed: %v", err)
	}
}

// Login

func TestLogin_Success(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	svc := service.AuthService{DB: db}

	sessionID, err := svc.Login("alice", "secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	svc := service.AuthService{DB: db}

	_, err := svc.Login("alice", "wrong")
	if !errors.Is(err, service.ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := service.AuthService{DB: db}

	_, err := svc.Login("nobody", "password")
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// Logout

func TestLogout_Success(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	sessionID := createTestSession(t, db, "alice")
	svc := service.AuthService{DB: db}

	if err := svc.Logout(sessionID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLogout_UnknownSession_Succeeds(t *testing.T) {
	db := newTestDB(t)
	svc := service.AuthService{DB: db}

	// DELETE is a no-op on missing rows, so logout is idempotent.
	if err := svc.Logout("01JMVNP7HZE8Q3K5X2WRT4Y6BD"); err != nil {
		t.Fatalf("expected no error for unknown session, got %v", err)
	}
}

// CheckAuth

func TestCheckAuth_ValidSession(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	sessionID := createTestSession(t, db, "alice")
	svc := service.AuthService{DB: db, SessionTTL: time.Hour}

	username, err := svc.CheckAuth(sessionID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", username)
	}
}

func TestCheckAuth_SessionNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := service.AuthService{DB: db, SessionTTL: time.Hour}

	_, err := svc.CheckAuth("01JMVNP7HZE8Q3K5X2WRT4Y6BD")
	if !errors.Is(err, service.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestCheckAuth_ExpiredSession(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	sessionID := createTestSession(t, db, "alice")
	backdateSession(t, db, sessionID, 2*time.Hour)
	svc := service.AuthService{DB: db, SessionTTL: time.Hour}

	_, err := svc.CheckAuth(sessionID)
	if !errors.Is(err, service.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestCheckAuth_ZeroTTL_NeverExpires(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	sessionID := createTestSession(t, db, "alice")
	backdateSession(t, db, sessionID, 365*24*time.Hour)
	svc := service.AuthService{DB: db, SessionTTL: 0}

	_, err := svc.CheckAuth(sessionID)
	if err != nil {
		t.Fatalf("expected no error with zero TTL, got %v", err)
	}
}

func TestCheckAuth_SlidingTTL_RecentLastUsedAt(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "alice", "secret")
	sessionID := createTestSession(t, db, "alice")
	backdateSession(t, db, sessionID, 2*time.Hour)    // created_at is outside TTL
	setSessionLastUsed(t, db, sessionID, 30*time.Minute) // but last_used_at is within TTL
	svc := service.AuthService{DB: db, SessionTTL: time.Hour}

	_, err := svc.CheckAuth(sessionID)
	if err != nil {
		t.Fatalf("expected no error for recently-used session, got %v", err)
	}
}
