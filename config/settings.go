package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds global and project-specific parameters
type Settings struct {
	TimeoutSeconds      int    `json:"timeout_seconds"`
	MaxRetries          int    `json:"max_retries"`
	DefaultOutputFolder string `json:"default_output_folder"`
	Debug               bool   `json:"debug"`
	VimMode             bool   `json:"vim_mode"`
}

const (
	DefaultTimeoutSeconds      = 60
	DefaultMaxRetries          = 3
	DefaultOutputFolderValue   = "./output"
)

// GetGlobalSettingsPath returns the global settings file path (e.g. ~/.synthspec/settings.json)
func GetGlobalSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".synthspec", "settings.json"), nil
}

// GetLocalSettingsPath returns the workspace-specific settings file path (e.g. .synthspec/settings.json)
func GetLocalSettingsPath() string {
	return filepath.Join(".synthspec", "settings.json")
}

// LoadSettings loads settings from global config, then overrides with local workspace config if present.
func LoadSettings() (*Settings, error) {
	// Initialize with default values
	s := &Settings{
		TimeoutSeconds:      DefaultTimeoutSeconds,
		MaxRetries:          DefaultMaxRetries,
		DefaultOutputFolder: DefaultOutputFolderValue,
		Debug:               false,
		VimMode:             false,
	}

	// 1. Try to load from global settings
	if globalPath, err := GetGlobalSettingsPath(); err == nil {
		mergeSettingsFromFile(s, globalPath)
	}

	// 2. Try to load from local settings
	mergeSettingsFromFile(s, GetLocalSettingsPath())

	return s, nil
}

func mergeSettingsFromFile(s *Settings, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}

	if loaded.TimeoutSeconds > 0 {
		s.TimeoutSeconds = loaded.TimeoutSeconds
	}
	if loaded.MaxRetries >= 0 {
		s.MaxRetries = loaded.MaxRetries
	}
	if loaded.DefaultOutputFolder != "" {
		s.DefaultOutputFolder = loaded.DefaultOutputFolder
	}
	
	// Only override debug/vim_mode if they are explicitly present in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["debug"]; ok {
			s.Debug = loaded.Debug
		}
		if _, ok := raw["vim_mode"]; ok {
			s.VimMode = loaded.VimMode
		}
	}
}

// SaveSettings persists settings to either the global path or the local path
func SaveSettings(s *Settings, global bool) error {
	var path string
	var err error

	if global {
		path, err = GetGlobalSettingsPath()
		if err != nil {
			return err
		}
	} else {
		path = GetLocalSettingsPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
