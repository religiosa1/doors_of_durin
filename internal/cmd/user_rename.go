package cmd

import (
	"fmt"

	"github.com/religiosa1/doors_of_durin/internal/repository/users"
)

type UserRename struct {
	CommonArgs `embed:""`
	NewName    string `short:"n" help:"New username"`
	Username   string `arg:"" optional:"" help:"Current username"`
}

func (u *UserRename) Run() error {
	username := u.Username
	newName := u.NewName

	if username == "" {
		var err error
		username, err = readLine("Current username: ")
		if err != nil {
			return err
		}
	}
	if newName == "" {
		var err error
		newName, err = readLine("New username: ")
		if err != nil {
			return err
		}
	}

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if newName == "" {
		return fmt.Errorf("new name cannot be empty")
	}

	_, db, err := loadConfigAndDB(u.Config)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := users.Rename(*db, username, newName); err != nil {
		return fmt.Errorf("renaming user: %w", err)
	}
	fmt.Printf("User %q renamed to %q.\n", username, newName)
	return nil
}
