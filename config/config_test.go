package config

import (
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
