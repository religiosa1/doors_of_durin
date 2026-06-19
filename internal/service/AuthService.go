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

// those granular errors are not exposed outside of service layer to the caller
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrBadPassword     = errors.New("wrong password")
	ErrUserDisabled    = errors.New("user account is disabled")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// LoginNoSession logs user in without issuing a session, but updates last
// basic login timestamp in DB -- for basic auth flow
func (s AuthService) LoginNoSession(username string, password string) error {
	_, err := s.checkPassword(username, password)
	if err != nil {
		return err
	}
	err = users.BumpLastBasicAuthTimestamp(*s.DB, username)
	if err != nil {
		return err
	}
	return nil
}

// Login logs a user in and issues a new Session -- for normal login flow
func (s AuthService) Login(username string, password string) (string, error) {
	userID, err := s.checkPassword(username, password)
	if err != nil {
		return "", err
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

func (s AuthService) checkPassword(username string, password string) (int64, error) {
	userID, err := users.CheckPassword(*s.DB, username, password)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return 0, ErrUserNotFound
		}
		if errors.Is(err, users.ErrNoPasswordSet) {
			return 0, ErrUserDisabled
		}
		if errors.Is(err, users.ErrBadPassword) {
			return 0, ErrBadPassword
		}
		return 0, err
	}
	return userID, nil
}
