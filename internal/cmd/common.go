package cmd

import (
	"fmt"

	"github.com/religiosa1/doors_of_durin/internal/config"
	"github.com/religiosa1/doors_of_durin/internal/repository"
)

type CommonArgs struct {
	Config string `short:"c" help:"Path to config file" placeholder:"config.yml"`
}

// MergeValueInto sets dst to src if src is non-zero.
func MergeValueInto[T comparable](dst *T, src T) {
	var zero T
	if src != zero {
		*dst = src
	}
}

func loadConfigAndDBForCli(configPath string) (config.Config, *repository.DB, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return cfg, nil, fmt.Errorf("loading config: %w", err)
	}
	// we're passing nil to migrator in CLI to avoid  potential migration messages in CLI usage
	db, err := repository.New(cfg.DBFile, nil)
	if err != nil {
		return cfg, nil, fmt.Errorf("opening database: %w", err)
	}
	return cfg, db, nil
}
