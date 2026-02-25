// Package sessions provides function to work with sessions in the DB
package sessions

import (
	"database/sql"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/religiosa1/auth_server/internal/repository"
)

// GetSession returns the session with the given ID, joining the users table to
// include the username. If notBefore is non-zero, sessions whose activity
// timestamp (COALESCE(last_used_at, created_at)) predates it are treated as
// expired and return repository.ErrRecordNotFound.
func GetSession(db repository.DB, sessionID string, notBefore time.Time) (Session, error) {
	query := `
		SELECT s.id, u.name AS username, s.created_at, s.last_used_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`
	args := []any{sessionID}
	if !notBefore.IsZero() {
		query += " AND COALESCE(s.last_used_at, s.created_at) >= ?"
		args = append(args, notBefore.UTC().Format("2006-01-02 15:04:05"))
	}
	var session Session
	err := db.DB.Get(&session, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return session, repository.ErrRecordNotFound
	}
	return session, err
}

func RegisterSessionUsage(db repository.DB, sessionID string) error {
	_, err := db.DB.Exec("UPDATE `sessions` SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?", sessionID)
	return err
}

func CreateSession(db repository.DB, userID int64) (string, error) {
	id := ulid.Make().String()
	query := "INSERT INTO `sessions` (id, user_id) VALUES (?, ?)"
	_, err := db.DB.Exec(query, id, userID)
	return id, err
}

func DeleteSession(db repository.DB, sessionID string) error {
	_, err := db.DB.Exec("DELETE FROM `sessions` WHERE id = ?", sessionID)
	return err
}

type Session struct {
	ID          string     `db:"id"           json:"id"`
	Username    string     `db:"username"     json:"username"`
	CreatedAt   time.Time  `db:"created_at"   json:"createdAt"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"lastUsedAt"`
}

type Filter struct {
	Username       string
	CreatedBefore  *time.Time
	LastUsedBefore *time.Time
}

func (f Filter) isEmpty() bool {
	return f.Username == "" && f.CreatedBefore == nil && f.LastUsedBefore == nil
}

func (f Filter) conditions() ([]string, []any) {
	var conds []string
	var args []any
	if f.Username != "" {
		conds = append(conds, "user_id = (SELECT id FROM users WHERE name = ?)")
		args = append(args, f.Username)
	}
	if f.CreatedBefore != nil {
		conds = append(conds, "s.created_at < ?")
		args = append(args, f.CreatedBefore.UTC().Format("2006-01-02 15:04:05"))
	}
	if f.LastUsedBefore != nil {
		conds = append(conds, "s.last_used_at < ?")
		args = append(args, f.LastUsedBefore.UTC().Format("2006-01-02 15:04:05"))
	}
	return conds, args
}

func List(db repository.DB, filter Filter) ([]Session, error) {
	conds, args := filter.conditions()
	query := `
		SELECT s.id, u.name AS username, s.created_at, s.last_used_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id`
	for i, c := range conds {
		if i == 0 {
			query += " WHERE " + c
		} else {
			query += " AND " + c
		}
	}
	query += " ORDER BY s.created_at DESC"

	var result []Session
	err := db.DB.Select(&result, query, args...)
	return result, err
}

func DeleteSessions(db repository.DB, filter Filter) (int64, error) {
	if filter.isEmpty() {
		return 0, errors.New("at least one filter must be specified to delete sessions")
	}

	// DELETE doesn't support JOIN, so rewrite username filter as subquery on sessions table directly
	var conds []string
	var args []any
	if filter.Username != "" {
		conds = append(conds, "user_id = (SELECT id FROM users WHERE name = ?)")
		args = append(args, filter.Username)
	}
	if filter.CreatedBefore != nil {
		conds = append(conds, "created_at < ?")
		args = append(args, filter.CreatedBefore.UTC().Format("2006-01-02 15:04:05"))
	}
	if filter.LastUsedBefore != nil {
		conds = append(conds, "last_used_at < ?")
		args = append(args, filter.LastUsedBefore.UTC().Format("2006-01-02 15:04:05"))
	}

	query := "DELETE FROM `sessions`"
	for i, c := range conds {
		if i == 0 {
			query += " WHERE " + c
		} else {
			query += " AND " + c
		}
	}

	result, err := db.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
