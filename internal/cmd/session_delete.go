package cmd

import (
	"fmt"
	"time"

	"github.com/religiosa1/doors_of_durin/internal/repository/sessions"
)

type sessionFilterArgs struct {
	Username       string `short:"u" help:"Filter by username"`
	CreatedBefore  string `short:"b" name:"created-before" help:"Filter sessions created before DATETIME (RFC3339 or 2006-01-02 15:04:05)"`
	LastUsedBefore string `short:"a" name:"last-used-before" help:"Filter sessions last used before DATETIME (RFC3339 or 2006-01-02 15:04:05)"`
}

func (f sessionFilterArgs) toFilter() (sessions.Filter, error) {
	filter := sessions.Filter{Username: f.Username}
	if f.CreatedBefore != "" {
		t, err := parseDateTime(f.CreatedBefore)
		if err != nil {
			return filter, fmt.Errorf("--created-before: %w", err)
		}
		filter.CreatedBefore = &t
	}
	if f.LastUsedBefore != "" {
		t, err := parseDateTime(f.LastUsedBefore)
		if err != nil {
			return filter, fmt.Errorf("--last-used-before: %w", err)
		}
		filter.LastUsedBefore = &t
	}
	return filter, nil
}

type SessionDelete struct {
	CommonArgs        `embed:""`
	sessionFilterArgs `embed:""`
}

func (s *SessionDelete) Run() error {
	filter, err := s.sessionFilterArgs.toFilter()
	if err != nil {
		return err
	}

	_, db, err := loadConfigAndDB(s.Config)
	if err != nil {
		return err
	}
	defer db.Close()

	n, err := sessions.DeleteSessions(*db, filter)
	if err != nil {
		return fmt.Errorf("deleting sessions: %w", err)
	}
	fmt.Printf("Deleted %d session(s).\n", n)
	return nil
}

func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as datetime (use RFC3339 or \"2006-01-02 15:04:05\")", s)
}
