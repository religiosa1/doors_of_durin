// Package sessions provides function to work with sessions in the DB
package sessions

import (
	"database/sql"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/religiosa1/auth_server/internal/repository"
)

func CheckSessionExists(db repository.DB, sessionID string) error {
	var id string
	err := db.DB.QueryRow("SELECT `id` FROM `sessions` WHERE `id` = ?", sessionID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrRecordNotFound
	}
	return err
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

func DeleteAllSessionsForUser(db repository.DB, userID int64) (int64, error) {
	result, err := db.DB.Exec("DELETE FROM `sessions` WHERE user_id = ?", userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func DeleteAllSessionsIssuedBeforeTime(db repository.DB, datetime time.Time) (int64, error) {
	// CURRENT_TIMESTAMP stores UTC in "2006-01-02 15:04:05" format; bind in the same
	// format so the string comparison in SQLite works correctly.
	result, err := db.DB.Exec(
		"DELETE FROM `sessions` WHERE created_at < ?",
		datetime.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
