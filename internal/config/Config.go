// Package config defines configuration structure and their parsing methods
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const configPathEnvKey = "CONFIG_PATH"

type Config struct {
	// logger minimum level: "debug", "info", "warn", "error"
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	// structured log output type: "json", "text"
	LogType    string        `yaml:"log_type" env:"LOG_TYPE" env-default:"json"`
	DBFile     string        `yaml:"db_file" env:"DB_FILE" env-default:"auth_server.sqlite3"`
	Port       string        `yaml:"port" env:"PORT" env-default:"4000"`
	Host       string        `yaml:"host" env:"HOST" env-default:"localhost"`
	SessionTTL time.Duration `yaml:"session_ttl" env:"SESSION_TTL" env-default:"168h"`
}

func Load(configPath string) (Config, error) {
	var triedPaths []string
	pathExplicitlySet := configPath != ""

	if !pathExplicitlySet {
		configPath = os.Getenv(configPathEnvKey)
		if configPath != "" {
			triedPaths = append(triedPaths, fmt.Sprintf("%s (from %s)", configPath, configPathEnvKey))
		}
		if configPath == "" {
			// Try default config paths in order: user config, then global config
			defaultPaths := getDefaultConfigPaths()
			for _, path := range defaultPaths {
				triedPaths = append(triedPaths, path)
				if _, err := os.Stat(path); err == nil {
					configPath = path
					break
				}
			}
		}
	} else {
		triedPaths = append(triedPaths, fmt.Sprintf("%s (explicitly set)", configPath))
	}

	var cfg Config
	fileExists := false
	if configPath != "" {
		_, err := os.Stat(configPath)
		fileExists = !os.IsNotExist(err)
	}

	if !fileExists {
		if pathExplicitlySet {
			return cfg, fmt.Errorf("specified config file does not exist: %s", configPath)
		}
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return cfg, fmt.Errorf("error loading configuration from environment: %w\nTried config paths:\n  %s",
				err, formatTriedPaths(triedPaths))
		}
	} else if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return cfg, fmt.Errorf("error loading configuration file: %w", err)
	}

	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return cfg, err
	}

	if err := validateLogType(cfg.LogType); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func validateLogLevel(logLevel string) error {
	switch logLevel {
	case "debug", "info", "warn", "error":
		return nil
	}
	return fmt.Errorf("unknown log level: '%s'", logLevel)
}

func validateLogType(logType string) error {
	switch logType {
	case "json", "text":
		return nil
	}
	return fmt.Errorf("unknown log type: %s", logType)
}

func formatTriedPaths(paths []string) string {
	if len(paths) == 0 {
		return "none"
	}
	return strings.Join(paths, "\n  ")
}
