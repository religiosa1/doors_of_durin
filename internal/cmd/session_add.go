package cmd

import (
	"fmt"
	"os"

	"github.com/religiosa1/doors_of_durin/internal/http/handlers"
	"github.com/religiosa1/doors_of_durin/internal/service"
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

	cfg, db, err := loadConfigAndDBForCli(s.Config)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	authService := service.AuthService{
		DB:         db,
		SessionTTL: cfg.SessionTTL,
	}

	sessionID, err := authService.Login(username, password)
	if err != nil {
		return err
	}

	fmt.Println(sessionID)
	fmt.Fprintf(os.Stderr, "Use this session in a cookie: %s=%s\n", handlers.SessionCookieName, sessionID)
	return nil
}
