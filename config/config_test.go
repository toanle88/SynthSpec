package config

import (
	"os"
	"testing"
)

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
		if bp.ID == "fintech-saas" {
			hasFintech = true
			if bp.Name == "" || bp.Description == "" || bp.Facts.Functional == "" {
				t.Errorf("fintech-saas blueprint fields should not be empty")
			}
		}
		if bp.ID == "internal-crud" {
			hasCrud = true
			if bp.Name == "" || bp.Description == "" || bp.Facts.Functional == "" {
				t.Errorf("internal-crud blueprint fields should not be empty")
			}
		}
	}

	if !hasFintech {
		t.Error("missing fintech-saas blueprint")
	}
	if !hasCrud {
		t.Error("missing internal-crud blueprint")
	}
}

func TestSettings(t *testing.T) {
	// Clean up any existing local settings for the duration of the test
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

	defer func() {
		if origExist {
			_ = os.WriteFile(localPath, origData, 0644)
		} else {
			_ = os.Remove(localPath)
		}
	}()

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	// Verify defaults
	if s.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected default timeout %d, got %d", DefaultTimeoutSeconds, s.TimeoutSeconds)
	}

	// Change values and save
	s.TimeoutSeconds = 120
	s.MaxRetries = 5
	s.DefaultOutputFolder = "./test_output"

	if err := SaveSettings(s, false); err != nil {
		t.Fatalf("failed to save local settings: %v", err)
	}

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
