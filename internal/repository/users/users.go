// Package users provide abstraction for accessing users in the database
package users

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/religiosa1/doors_of_durin/internal/repository"
)

// ErrNoPasswordSet is returned when a user account has no password hash stored.
var ErrNoPasswordSet = errors.New("user has no password set")

type User struct {
	Name       string    `db:"name"        json:"name"`
	CreatedAt  time.Time `db:"created_at"  json:"createdAt"`
	ModifiedAt time.Time `db:"modified_at" json:"modifiedAt"`
}

func List(db repository.DB) ([]User, error) {
	var result []User
	err := db.DB.Select(&result, "SELECT name, created_at, modified_at FROM users ORDER BY name")
	return result, err
}

func GetUserID(db repository.DB, username string) (int64, error) {
	var id int64
	err := db.DB.QueryRow("SELECT `id` FROM `users` WHERE `name` = ?", username).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, repository.ErrRecordNotFound
	}
	return id, err
}

func Create(db repository.DB, username string, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = db.DB.Exec(
		"INSERT INTO `users` (`name`, `password_hash`) VALUES (?, ?)",
		username, hash,
	)
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		return repository.ErrUniqueConstraint
	}
	return err
}

func CheckPassword(db repository.DB, username string, password string) (int64, error) {
	var userID int64
	var hash sql.NullString
	err := db.DB.QueryRow("SELECT `id`, `password_hash` FROM `users` WHERE `name` = ?", username).Scan(&userID, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, repository.ErrRecordNotFound
	}
	if err != nil {
		return 0, err
	}
	if !hash.Valid {
		return 0, ErrNoPasswordSet
	}

	if !checkPassword(password, hash.String) {
		return 0, nil
	}
	return userID, nil
}

func UpdatePassword(db repository.DB, username string, newPassword string) error {
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	result, err := db.DB.Exec(
		"UPDATE `users` SET `password_hash` = ?, `modified_at` = CURRENT_TIMESTAMP WHERE `name` = ?",
		hash, username,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrRecordNotFound
	}
	return nil
}

func Rename(db repository.DB, oldUserName string, newUserName string) error {
	result, err := db.DB.Exec(
		"UPDATE `users` SET `name` = ?, `modified_at` = CURRENT_TIMESTAMP WHERE `name` = ?",
		newUserName, oldUserName,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return repository.ErrUniqueConstraint
		}
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrRecordNotFound
	}
	return nil
}

// Delete deletes a user by username.
func Delete(db repository.DB, username string) error {
	result, err := db.DB.Exec("DELETE FROM `users` WHERE `name` = ?", username)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrRecordNotFound
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
