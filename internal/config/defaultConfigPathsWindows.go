//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// These variables can be overridden at build time using ldflags.
// UserConfigPath: path template for user-specific config, use ${APPDATA} as placeholder
// GlobalConfigPath: path template for global config, use ${PROGRAMDATA} as placeholder
var (
	UserConfigPath   = "${APPDATA}\\durin\\config.yml"
	GlobalConfigPath = "${PROGRAMDATA}\\durin\\config.yml"
)

// getDefaultConfigPaths returns config paths in priority order:
// 1. User config: ${APPDATA}\durin\config.yml
// 2. Global config: ${PROGRAMDATA}\durin\config.yml
func getDefaultConfigPaths() []string {
	paths := make([]string, 0, 2)

	// User config path - expand environment variables
	userPath := os.ExpandEnv(UserConfigPath)
	if userPath != "" {
		paths = append(paths, filepath.FromSlash(userPath))
	}

	// Global config path - expand environment variables
	globalPath := os.ExpandEnv(GlobalConfigPath)
	if globalPath != "" {
		paths = append(paths, filepath.FromSlash(globalPath))
	}

	return paths
}
