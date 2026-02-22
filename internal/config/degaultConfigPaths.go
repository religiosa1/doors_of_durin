//go:build unix

package config

import (
	"os"
	"path/filepath"
)

// These variables can be overridden at build time using ldflags.
// UserConfigPath: path template for user-specific config, use ${XDG_CONFIG_HOME} or ${HOME} as placeholders
// GlobalConfigPath: path to global config file
var (
	UserConfigPath   = "${XDG_CONFIG_HOME}/auth_server/config.yml"
	GlobalConfigPath = "/etc/auth_server.yml"
)

// getDefaultConfigPaths returns config paths in priority order:
// 1. User config: $XDG_CONFIG_HOME/auth_server/config.yml (or ~/.config/auth_server/config.yml)
// 2. Global config: /etc/auth_server.yml
func getDefaultConfigPaths() []string {
	paths := make([]string, 0, 2)

	// User config path - expand environment variables
	userPath := UserConfigPath

	// Replace placeholders
	userPath = os.Expand(userPath, func(key string) string {
		switch key {
		case "XDG_CONFIG_HOME":
			return getXdgConfigHome()
		default:
			return os.Getenv(key)
		}
	})
	if userPath != "" {
		paths = append(paths, userPath)
	}

	// Global config path
	globalPath := os.ExpandEnv(GlobalConfigPath)
	if globalPath != "" {
		paths = append(paths, globalPath)
	}

	return paths
}

// Expanding XDG_CONFIG_HOME as described in https://specifications.freedesktop.org/basedir/latest/#variables
func getXdgConfigHome() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome
	}

	// XDG Base Directory spec: default is $HOME/.config
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return ""
	}
	return filepath.Join(homeDir, ".config")
}
