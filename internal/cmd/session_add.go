package cmd

import (
	"fmt"
	"os"

	"github.com/religiosa1/auth_server/internal/http/handlers"
	"github.com/religiosa1/auth_server/internal/repository"
	"github.com/religiosa1/auth_server/internal/repository/sessions"
	"github.com/religiosa1/auth_server/internal/repository/users"
)

type SessionAdd struct {
	CommonArgs `embed:""`
	Password   string `short:"p" help:"User password"`
	Username   string `arg:"" optional:"" help:"Username"`
}

func (s *SessionAdd) Run() error {
	username := s.Username
	password := s.Password

	if username == "" {
		var err error
		username, err = readLine("Username: ")
		if err != nil {
			return err
		}
	}
	if password == "" {
		var err error
		password, err = readPassword("Password: ")
		if err != nil {
			return err
		}
	}

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	_, db, err := loadConfigAndDB(s.Config)
	if err != nil {
		return err
	}
	defer db.Close()

	ok, err := users.CheckPassword(*db, username, password)
	if err != nil {
		if err == repository.ErrRecordNotFound {
			return fmt.Errorf("user %q not found", username)
		}
		return err
	}
	if !ok {
		return fmt.Errorf("wrong password for user %q", username)
	}

	userID, err := users.GetUserID(*db, username)
	if err != nil {
		return err
	}

	sessionID, err := sessions.CreateSession(*db, userID)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	fmt.Println(sessionID)
	fmt.Fprintf(os.Stderr, "Use this session in a cookie: %s=%s\n", handlers.SessionCookieName, sessionID)
	return nil
}
