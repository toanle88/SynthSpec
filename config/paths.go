package config

import (
	"os"
	"path/filepath"
)

// GetSynthspecRoot returns the base directory for SynthSpec data.
// It prefers the user's config directory with a fallback to the current working directory.
func GetSynthspecRoot() string {
	if envRoot := os.Getenv("SYNTHSPEC_ROOT"); envRoot != "" {
		return envRoot
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "synthspec")
	}
	return "synthspec"
}
