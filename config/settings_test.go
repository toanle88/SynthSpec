package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(".synthspec")
	t.Setenv("HOME", tmpDir)
	if _, err := GetGlobalSettingsPath(); err == nil {
		os.RemoveAll(filepath.Join(tmpDir, ".synthspec"))
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings with no files should return defaults: %v", err)
	}
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected timeout %d, got %d", DefaultTimeoutSeconds, s.TimeoutSeconds)
	}
	if s.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected max retries %d, got %d", DefaultMaxRetries, s.MaxRetries)
	}
	if s.DefaultOutputFolder != DefaultOutputFolderValue {
		t.Errorf("expected output folder %q, got %q", DefaultOutputFolderValue, s.DefaultOutputFolder)
	}
	if s.Debug {
		t.Errorf("expected Debug=false by default")
	}
}

func TestSaveAndLoadSettings_Local(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpDir)

	s := &Settings{
		TimeoutSeconds:      60,
		MaxRetries:          3,
		DefaultOutputFolder: "./custom_out",
		Debug:               true,
		HardBudgetCap:       5.50,
	}

	if err := SaveSettings(s, false); err != nil {
		t.Fatalf("SaveSettings local failed: %v", err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save failed: %v", err)
	}
	if loaded.TimeoutSeconds != 60 {
		t.Errorf("expected timeout 60, got %d", loaded.TimeoutSeconds)
	}
	if loaded.MaxRetries != 3 {
		t.Errorf("expected retries 3, got %d", loaded.MaxRetries)
	}
	if loaded.DefaultOutputFolder != "./custom_out" {
		t.Errorf("expected folder %q, got %q", "./custom_out", loaded.DefaultOutputFolder)
	}
	if !loaded.Debug {
		t.Errorf("expected Debug=true")
	}
	if loaded.HardBudgetCap != 5.50 {
		t.Errorf("expected HardBudgetCap 5.50, got %f", loaded.HardBudgetCap)
	}
}

func TestLoadSettings_LocalOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(tmpDir, "homeuser")
	os.MkdirAll(homeDir, 0755)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)

	globalDir := filepath.Join(homeDir, ".synthspec")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalData := `{"timeout_seconds": 500, "max_retries": 20, "default_output_folder": "./global_out", "debug": true}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.json"), []byte(globalData), 0644); err != nil {
		t.Fatal(err)
	}

	localDir := ".synthspec"
	os.MkdirAll(localDir, 0755)
	localData := `{"timeout_seconds": 100, "max_retries": 5}`
	if err := os.WriteFile(filepath.Join(localDir, "settings.json"), []byte(localData), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}
	if loaded.TimeoutSeconds != 100 {
		t.Errorf("expected local timeout 100, got %d", loaded.TimeoutSeconds)
	}
	if loaded.MaxRetries != 5 {
		t.Errorf("expected local retries 5, got %d", loaded.MaxRetries)
	}
	if !loaded.Debug {
		t.Errorf("expected Debug=true from global (not overridden)")
	}
}

func TestMergeSettingsFromFile(t *testing.T) {
	s := &Settings{
		TimeoutSeconds:      DefaultTimeoutSeconds,
		MaxRetries:          DefaultMaxRetries,
		DefaultOutputFolder: DefaultOutputFolderValue,
	}

	mergeSettingsFromFile(s, "nonexistent.json")
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("non-existent file should not modify settings")
	}

	tmpFile := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(tmpFile, []byte(`{bad json`), 0644)
	mergeSettingsFromFile(s, tmpFile)
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("malformed JSON should not modify settings")
	}

	validFile := filepath.Join(t.TempDir(), "good.json")
	os.WriteFile(validFile, []byte(`{"timeout_seconds": 99}`), 0644)
	mergeSettingsFromFile(s, validFile)
	if s.TimeoutSeconds != 99 {
		t.Errorf("expected timeout 99, got %d", s.TimeoutSeconds)
	}
	if s.MaxRetries != 0 {
		t.Errorf("expected MaxRetries 0, got %d", s.MaxRetries)
	}
}

func setupTestSettingsEnvironment(_ *testing.T) (string, string, bool, []byte, bool, []byte) {
	localPath := GetLocalSettingsPath()
	origExist := false
	var origData []byte
	if _, err := os.Stat(localPath); err == nil {
		origExist = true
		if data, err := os.ReadFile(localPath); err == nil {
			origData = data
		}
		_ = os.Remove(localPath)
	}

	globalPath, gErr := GetGlobalSettingsPath()
	origGlobalExist := false
	var origGlobalData []byte
	if gErr == nil {
		if _, err := os.Stat(globalPath); err == nil {
			origGlobalExist = true
			if data, err := os.ReadFile(globalPath); err == nil {
				origGlobalData = data
			}
			_ = os.Remove(globalPath)
		}
	}

	return localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData
}

func restoreTestSettingsEnvironment(localPath, globalPath string, origExist bool, origData []byte, origGlobalExist bool, origGlobalData []byte) {
	if origExist {
		_ = os.WriteFile(localPath, origData, 0644)
	} else {
		_ = os.Remove(localPath)
	}
	if origGlobalExist {
		_ = os.WriteFile(globalPath, origGlobalData, 0644)
	}
}

func verifyLoadedSettings(t *testing.T) {
	s2, err := LoadSettings()
	if err != nil {
		t.Fatalf("failed to load modified settings: %v", err)
	}

	if s2.TimeoutSeconds != 120 {
		t.Errorf("expected loaded timeout 120, got %d", s2.TimeoutSeconds)
	}
	if s2.MaxRetries != 5 {
		t.Errorf("expected loaded max retries 5, got %d", s2.MaxRetries)
	}
	if s2.DefaultOutputFolder != "./test_output" {
		t.Errorf("expected loaded default output folder './test_output', got %s", s2.DefaultOutputFolder)
	}
}

func TestSettings(t *testing.T) {
	localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData := setupTestSettingsEnvironment(t)
	defer restoreTestSettingsEnvironment(localPath, globalPath, origExist, origData, origGlobalExist, origGlobalData)

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected default timeout %d, got %d", DefaultTimeoutSeconds, s.TimeoutSeconds)
	}

	s.TimeoutSeconds = 120
	s.MaxRetries = 5
	s.DefaultOutputFolder = "./test_output"

	if err := SaveSettings(s, false); err != nil {
		t.Fatalf("failed to save local settings: %v", err)
	}

	verifyLoadedSettings(t)
}
