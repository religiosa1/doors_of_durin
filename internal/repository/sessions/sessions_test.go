package sessions_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/repository"
	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
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

// mustCreateUser inserts a user directly, bypassing password hashing, and returns their ID.
func mustCreateUser(t *testing.T, db *repository.DB) int64 {
	t.Helper()
	result, err := db.DB.Exec("INSERT INTO `users` (`name`) VALUES (?)", "testuser")
	if err != nil {
		t.Fatalf("mustCreateUser: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("mustCreateUser LastInsertId: %v", err)
	}
	return id
}

func TestCreateSession(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestGetSession(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	session, err := sessions.GetSession(*db, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session.ID != sessionID {
		t.Fatalf("expected session ID %q, got %q", sessionID, session.ID)
	}
	if session.Username != "testuser" {
		t.Fatalf("expected username %q, got %q", "testuser", session.Username)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := sessions.GetSession(*db, "nonexistent")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRegisterSessionUsage(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := sessions.RegisterSessionUsage(*db, sessionID); err != nil {
		t.Fatalf("RegisterSessionUsage: %v", err)
	}

	var lastUsedAt sql.NullTime
	err = db.DB.QueryRow("SELECT last_used_at FROM sessions WHERE id = ?", sessionID).Scan(&lastUsedAt)
	if err != nil {
		t.Fatalf("querying last_used_at: %v", err)
	}
	if !lastUsedAt.Valid {
		t.Fatal("expected last_used_at to be set after RegisterSessionUsage")
	}
}

func TestDeleteSession(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := sessions.DeleteSession(*db, sessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, err = sessions.GetSession(*db, sessionID)
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected session to be gone, got %v", err)
	}
}

func TestDeleteSessions_ByUser(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	if _, err := sessions.CreateSession(*db, userID); err != nil {
		t.Fatalf("CreateSession 1: %v", err)
	}
	if _, err := sessions.CreateSession(*db, userID); err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	n, err := sessions.DeleteSessions(*db, sessions.Filter{Username: "testuser"})
	if err != nil {
		t.Fatalf("DeleteSessions: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 deleted, got %d", n)
	}
}

func TestDeleteSessions_ByCreatedBefore(t *testing.T) {
	db := newTestDB(t)
	userID := mustCreateUser(t, db)

	if _, err := sessions.CreateSession(*db, userID); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	t.Run("cutoff before session creation", func(t *testing.T) {
		cutoff := time.Now().Add(-time.Hour)
		n, err := sessions.DeleteSessions(*db, sessions.Filter{CreatedBefore: &cutoff})
		if err != nil {
			t.Fatalf("DeleteSessions: %v", err)
		}
		if n != 0 {
			t.Fatalf("expected 0 deleted, got %d", n)
		}
	})

	t.Run("cutoff after session creation", func(t *testing.T) {
		cutoff := time.Now().Add(time.Minute)
		n, err := sessions.DeleteSessions(*db, sessions.Filter{CreatedBefore: &cutoff})
		if err != nil {
			t.Fatalf("DeleteSessions: %v", err)
		}
		if n != 1 {
			t.Fatalf("expected 1 deleted, got %d", n)
		}
	})
}

func TestDeleteSessions_NoFilter(t *testing.T) {
	db := newTestDB(t)
	_, err := sessions.DeleteSessions(*db, sessions.Filter{})
	if err == nil {
		t.Fatal("expected error when no filter provided, got nil")
	}
}
