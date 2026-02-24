// Package users provide abstraction for accessing users in the database
package users

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/religiosa1/auth_server/internal/repository"
)

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

func CheckPassword(db repository.DB, username string, password string) (bool, error) {
	var hash sql.NullString
	err := db.DB.QueryRow("SELECT `password_hash` FROM `users` WHERE `name` = ?", username).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, repository.ErrRecordNotFound
	}
	if err != nil {
		return false, err
	}
	if !hash.Valid {
		return false, nil
	}
	return checkPassword(password, hash.String), nil
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
