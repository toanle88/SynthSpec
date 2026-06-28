package config

import (
	"os"
	"testing"
)

func assertBlueprintValid(t *testing.T, bp Blueprint) {
	if bp.Name == "" || bp.Description == "" || bp.Facts.Functional == "" {
		t.Errorf("%s blueprint fields should not be empty", bp.ID)
	}
}

func TestLoadBlueprints(t *testing.T) {
	blueprints, err := LoadBlueprints()
	if err != nil {
		t.Fatalf("failed to load blueprints: %v", err)
	}

	if len(blueprints) < 2 {
		t.Errorf("expected at least 2 default blueprints, got %d", len(blueprints))
	}

	var hasFintech, hasCrud bool
	for _, bp := range blueprints {
		switch bp.ID {
		case "fintech-saas":
			hasFintech = true
			assertBlueprintValid(t, bp)
		case "internal-crud":
			hasCrud = true
			assertBlueprintValid(t, bp)
		}
	}

	if !hasFintech {
		t.Error("missing fintech-saas blueprint")
	}
	if !hasCrud {
		t.Error("missing internal-crud blueprint")
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
