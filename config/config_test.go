package config

import (
	"os"
	"testing"
)

func TestLoadStandards_Embedded(t *testing.T) {
	standards, err := LoadStandards()
	if err != nil {
		t.Fatalf("LoadStandards embedded should succeed: %v", err)
	}
	if len(standards) == 0 {
		t.Fatal("expected at least one standard from embedded defaults")
	}
}

func TestLoadStandards_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`standards:
  - id: custom-test
    name: Custom Test Standard
    description: A test standard
    target_files: ["01_domain_model_use_cases.md"]
    criteria: "Must pass"
    min_score: 80
  - id: domain_model_purity
    name: Overridden Purity Standard
    description: Custom description for purity standard
    target_files: ["01_domain_model_use_cases.md"]
    criteria: "Custom criteria"
    min_score: 95`)
	if err := os.WriteFile("standards.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	standards, err := LoadStandards()
	if err != nil {
		t.Fatalf("LoadStandards with override should succeed: %v", err)
	}
	
	// Default standards + 1 new standard
	if len(standards) <= 1 {
		t.Errorf("expected merged standards to contain default + new, got %d", len(standards))
	}

	var foundCustom, foundOverridden bool
	for _, std := range standards {
		if std.ID == "custom-test" {
			foundCustom = true
		}
		if std.ID == "domain_model_purity" {
			foundOverridden = true
			if std.Name != "Overridden Purity Standard" {
				t.Errorf("expected overridden standard Name to be 'Overridden Purity Standard', got %q", std.Name)
			}
		}
	}

	if !foundCustom {
		t.Error("expected custom standard to be appended")
	}
	if !foundOverridden {
		t.Error("expected default standard to be overridden")
	}
}

func TestLoadTemplates_Embedded(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates embedded should succeed: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("expected at least one template from embedded defaults")
	}
}

func TestLoadTemplates_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`templates:
  - file_name: "custom_template.md"
    name: "Custom Template"
    prompt: "Generate a custom document"
  - file_name: "01_domain_model_use_cases.md"
    name: "Overridden Domain Template"
    prompt: "Overridden prompt"`)
	if err := os.WriteFile("templates.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates with override should succeed: %v", err)
	}

	if len(templates) <= 1 {
		t.Errorf("expected merged templates to contain default + new, got %d", len(templates))
	}

	var foundCustom, foundOverridden bool
	for _, tmpl := range templates {
		if tmpl.FileName == "custom_template.md" {
			foundCustom = true
		}
		if tmpl.FileName == "01_domain_model_use_cases.md" {
			foundOverridden = true
			if tmpl.Name != "Overridden Domain Template" {
				t.Errorf("expected overridden template Name to be 'Overridden Domain Template', got %q", tmpl.Name)
			}
		}
	}

	if !foundCustom {
		t.Error("expected custom template to be appended")
	}
	if !foundOverridden {
		t.Error("expected default template to be overridden")
	}
}

func TestLoadBlueprints_Override(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	overrideContent := []byte(`blueprints:
  - id: custom-bp
    name: Custom Blueprint
    description: A test blueprint
    facts:
      functional: "Custom functional"
      structural: "Custom structural"
      security: "Custom security"
      compliance: "Custom compliance"`)
	if err := os.WriteFile("blueprints.yaml", overrideContent, 0644); err != nil {
		t.Fatal(err)
	}

	blueprints, err := LoadBlueprints()
	if err != nil {
		t.Fatalf("LoadBlueprints with override should succeed: %v", err)
	}
	if len(blueprints) != 1 || blueprints[0].ID != "custom-bp" {
		t.Errorf("expected 1 custom blueprint, got %d", len(blueprints))
	}
}

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
