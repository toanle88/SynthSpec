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

	t.Run("Valid JSON and YAML block", func(t *testing.T) {
		content := `
# OpenAPI Specification
Here is the JSON specification:
` + "```json" + `
{
  "openapi": "3.0.0",
  "info": {
    "title": "SynthSpec API",
    "version": "1.0.0"
  }
}
` + "```" + `
And the yaml:
` + "```yaml" + `
paths:
  /users:
    get:
      summary: List users
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content)
		if err != nil {
			t.Errorf("expected valid markdown with code blocks to pass, got error: %v", err)
		}
	})

	t.Run("Malformed JSON block", func(t *testing.T) {
		content := `
# OpenAPI
` + "```json" + `
{
  "openapi": "3.0.0",
  "info": {
    "title": "SynthSpec API", // trailing comma or invalid comment
  }
}
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content)
		if err == nil {
			t.Error("expected syntax error on malformed JSON block, got nil")
		}
	})

	t.Run("Malformed YAML block", func(t *testing.T) {
		content := `
# YAML
` + "```yaml" + `
paths:
  /users: "unclosed string
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content)
		if err == nil {
			t.Error("expected syntax error on malformed YAML block, got nil")
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

	report := GenerateComplianceReport("TestProject", audits, stds, nil)
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
