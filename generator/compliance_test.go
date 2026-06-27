package generator

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func TestPerformStaticValidation(t *testing.T) {
	t.Run("Valid non-empty Markdown", func(t *testing.T) {
		content := "# API Guide"
		err := PerformStaticValidation("04_api_architecture_integration.md", content)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("Invalid empty Markdown", func(t *testing.T) {
		content := "   "
		err := PerformStaticValidation("04_api_architecture_integration.md", content)
		if err == nil {
			t.Error("expected empty content error, got nil")
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
