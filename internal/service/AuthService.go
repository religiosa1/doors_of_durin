// Package service wraps repo methods into consumption ready methods encapsulating the logic
package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/repository"
	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
	"github.com/religiosa1/doors_of_durin/internal/repository/users"
)

type AuthService struct {
	DB         *repository.DB
	SessionTTL time.Duration
}

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrWrongPassword   = errors.New("wrong password")
	ErrUserDisabled    = errors.New("user account is disabled")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

func (s AuthService) Login(username string, password string) (string, error) {
	userID, err := users.CheckPassword(*s.DB, username, password)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		if errors.Is(err, users.ErrNoPasswordSet) {
			return "", ErrUserDisabled
		}
		return "", err
	}
	if userID == 0 {
		return "", ErrWrongPassword
	}

	sessionID, err := sessions.CreateSession(*s.DB, userID)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

// Logout logs a user out. Please notice it never returns ErrSessionNotFound: Logout is idempotent
func (s AuthService) Logout(sessionID string) error {
	// repository won't return record not found either on delete
	return sessions.DeleteSession(*s.DB, sessionID)
}

// CheckAuth checks session is valid and register its usage in the db
// Returns username and potential error
func (s AuthService) CheckAuth(sessionID string) (string, error) {
	session, err := sessions.GetSession(*s.DB, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return "", ErrSessionNotFound
		}
		return "", fmt.Errorf("unexpected error while getting session from db %w", err)
	}

	if s.SessionTTL > 0 {
		activity := session.CreatedAt
		if session.LastUsedAt != nil && session.LastUsedAt.Compare(session.CreatedAt) > 0 {
			activity = *session.LastUsedAt
		}
		if time.Since(activity) > s.SessionTTL {
			return "", ErrSessionExpired
		}
	}

	err = sessions.RegisterSessionUsage(*s.DB, sessionID)
	if err != nil {
		return "", fmt.Errorf("error registering session usage: %w", err)
	}

	return session.Username, nil
}
