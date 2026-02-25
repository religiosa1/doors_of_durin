package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/religiosa1/auth_server/internal/repository/users"
)

type UserList struct {
	CommonArgs `embed:""`
	JSON       bool `short:"j" name:"json" help:"Output in JSON format"`
}

func (u *UserList) Run() error {
	_, db, err := loadConfigAndDB(u.Config)
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
	fmt.Fprintln(w, "NAME\tCREATED AT\tMODIFIED AT")
	for _, user := range list {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			user.Name,
			user.CreatedAt.Format(time.DateTime),
			user.ModifiedAt.Format(time.DateTime),
		)
	}
	return w.Flush()
}
