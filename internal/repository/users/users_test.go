package users_test

import (
	"errors"
	"testing"

	"github.com/religiosa1/doors_of_durin/internal/repository"
	"github.com/religiosa1/doors_of_durin/internal/repository/users"
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

func TestCreate(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "password123"); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "password"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	err := users.Create(*db, "alice", "other")
	if !errors.Is(err, repository.ErrUniqueConstraint) {
		t.Fatalf("expected ErrUniqueConstraint, got %v", err)
	}
}

func TestGetUserID(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "password"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, err := users.GetUserID(*db, "alice")
	if err != nil {
		t.Fatalf("GetUserID: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
}

func TestGetUserID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := users.GetUserID(*db, "nonexistent")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestCheckPassword(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "correct"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		userID, err := users.CheckPassword(*db, "alice", "correct")
		if err != nil {
			t.Fatalf("CheckPassword: %v", err)
		}
		if userID == 0 {
			t.Fatal("expected true for correct password")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		userID, err := users.CheckPassword(*db, "alice", "wrong")
		if err != nil {
			t.Fatalf("CheckPassword: %v", err)
		}
		if userID != 0 {
			t.Fatal("expected false for wrong password")
		}
	})
}

func TestCheckPassword_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := users.CheckPassword(*db, "nonexistent", "password")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "old"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := users.UpdatePassword(*db, "alice", "new"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	userID, err := users.CheckPassword(*db, "alice", "new")
	if err != nil {
		t.Fatalf("CheckPassword new: %v", err)
	}
	if userID == 0 {
		t.Fatal("expected new password to work after update")
	}

	userID, err = users.CheckPassword(*db, "alice", "old")
	if err != nil {
		t.Fatalf("CheckPassword old: %v", err)
	}
	if userID != 0 {
		t.Fatal("expected old password to fail after update")
	}
}

func TestUpdatePassword_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := users.UpdatePassword(*db, "nonexistent", "password")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRename(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "password"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := users.Rename(*db, "alice", "bob"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	if _, err := users.GetUserID(*db, "bob"); err != nil {
		t.Fatalf("GetUserID after rename: %v", err)
	}
	_, err := users.GetUserID(*db, "alice")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected old name to be gone, got %v", err)
	}
}

func TestRename_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := users.Rename(*db, "nonexistent", "bob")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRename_DuplicateName(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "pw"); err != nil {
		t.Fatalf("Create alice: %v", err)
	}
	if err := users.Create(*db, "bob", "pw"); err != nil {
		t.Fatalf("Create bob: %v", err)
	}
	err := users.Rename(*db, "alice", "bob")
	if !errors.Is(err, repository.ErrUniqueConstraint) {
		t.Fatalf("expected ErrUniqueConstraint, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	db := newTestDB(t)
	if err := users.Create(*db, "alice", "password"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := users.Delete(*db, "alice"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := users.GetUserID(*db, "alice")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected user to be gone, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := users.Delete(*db, "nonexistent")
	if !errors.Is(err, repository.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}
