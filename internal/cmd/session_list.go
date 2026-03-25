package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
)

type SessionList struct {
	CommonArgs        `embed:""`
	sessionFilterArgs `embed:""`
	JSON              bool `short:"j" name:"json" help:"Output in JSON format"`
}

func (s *SessionList) Run() error {
	filter, err := s.toFilter()
	if err != nil {
		return err
	}

	_, db, err := loadConfigAndDBForCli(s.Config)
	if err != nil {
		return err
	}
	defer db.Close()

	list, err := sessions.List(*db, filter)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	if s.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, err = fmt.Fprintln(w, "ID\tUSERNAME\tCREATED AT\tLAST USED AT")
	if err != nil {
		return err
	}
	for _, sess := range list {
		lastUsed := "-"
		if sess.LastUsedAt != nil {
			lastUsed = sess.LastUsedAt.Format(time.DateTime)
		}
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			sess.ID,
			sess.Username,
			sess.CreatedAt.Format(time.DateTime),
			lastUsed,
		)
		if err != nil {
			return err
		}
	}
	return w.Flush()
}
