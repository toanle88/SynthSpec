package generator

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func TestPerformStaticValidation(t *testing.T) {
	t.Run("Valid OpenAPI YAML", func(t *testing.T) {
		validYAML := "openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0"
		err := PerformStaticValidation("04_openapi_contract.yaml", validYAML)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("Invalid OpenAPI YAML", func(t *testing.T) {
		invalidYAML := "openapi: : 3.0.0"
		err := PerformStaticValidation("04_openapi_contract.yaml", invalidYAML)
		if err == nil {
			t.Error("expected syntax error, got nil")
		}
	})

	t.Run("Valid Backlog JSON", func(t *testing.T) {
		validJSON := `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TS-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`
		err := PerformStaticValidation("05_engineering_backlog.json", validJSON)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("Invalid Backlog JSON", func(t *testing.T) {
		invalidJSON := `{"epics": []}`
		err := PerformStaticValidation("05_engineering_backlog.json", invalidJSON)
		if err == nil {
			t.Error("expected error for empty epics backlog, got nil")
		}
	})
}

func TestGenerateComplianceReport(t *testing.T) {
	stds := []config.Standard{
		{
			ID:          "clean_architecture",
			Name:        "Clean Architecture",
			Description: "separation of concern",
			TargetFiles: []string{"02_system_architecture.md"},
			MinScore:    70,
		},
	}

	audits := []FileCompliance{
		{
			FileName: "02_system_architecture.md",
			Results: []gateway.ComplianceResult{
				{
					StandardID: "clean_architecture",
					Score:      80,
					Compliant:  true,
					Feedback:   "Good separation.",
				},
			},
			Err: nil,
		},
	}

	report := GenerateComplianceReport("TestProject", audits, stds)
	if !strings.Contains(report, "Clean Architecture") {
		t.Errorf("expected report to contain 'Clean Architecture'")
	}
	if !strings.Contains(report, "🟢 Compliant") {
		t.Errorf("expected report to indicate Compliant status")
	}
	if !strings.Contains(report, "80%") {
		t.Errorf("expected report to contain score 80%%")
	}
}
