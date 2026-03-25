package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/repository/users"
)

type UserList struct {
	CommonArgs `embed:""`
	JSON       bool `short:"j" name:"json" help:"Output in JSON format"`
}

func (u *UserList) Run() error {
	_, db, err := loadConfigAndDBForCli(u.Config)
	if err != nil {
		return err
	}
	defer db.Close()

	list, err := users.List(*db)
	if err != nil {
		return fmt.Errorf("listing users: %w", err)
	}

	if u.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, err = fmt.Fprintln(w, "NAME\tCREATED AT\tMODIFIED AT")
	if err != nil {
		return err
	}
	for _, user := range list {
		_, err := fmt.Fprintf(w, "%s\t%s\t%s\n",
			user.Name,
			user.CreatedAt.Format(time.DateTime),
			user.ModifiedAt.Format(time.DateTime),
		)
		if err != nil {
			return err
		}
	}
	return w.Flush()
}
