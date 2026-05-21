package cmd

import (
	"fmt"

	"github.com/religiosa1/doors_of_durin/internal/repository/users"
)

type UserDelete struct {
	CommonArgs `embed:""`
	Username   string `arg:"" optional:"" help:"Username to delete"`
}

func (u *UserDelete) Run() error {
	username := u.Username

	if username == "" {
		var err error
		username, err = readLine("Username: ")
		if err != nil {
			return err
		}
	}

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	ok, err := confirm(fmt.Sprintf("Delete user %q?", username))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Aborted.")
		return nil
	}

	_, db, err := loadConfigAndDBForCli(u.Config)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	if err := users.Delete(*db, username); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	fmt.Printf("User %q deleted.\n", username)
	return nil
}
