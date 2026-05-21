package cmd

import (
	"fmt"

	"github.com/religiosa1/doors_of_durin/internal/repository/users"
)

type UserAdd struct {
	CommonArgs `embed:""`
	Password   string `short:"p" help:"User password"`
	Username   string `arg:"" optional:"" help:"Username to add"`
}

func (u *UserAdd) Run() error {
	username := u.Username
	password := u.Password

	if username == "" || password == "" {
		var err error
		if username == "" {
			username, err = readLine("Username: ")
			if err != nil {
				return err
			}
		}
		if password == "" {
			password, err = readPassword("Password: ")
			if err != nil {
				return err
			}
			confirm, err := readPassword("Confirm password: ")
			if err != nil {
				return err
			}
			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}
		}
	}

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	_, db, err := loadConfigAndDBForCli(u.Config)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	if err := users.Create(*db, username, password); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	fmt.Printf("User %q created.\n", username)
	return nil
}
