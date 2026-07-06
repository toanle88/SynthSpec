package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func TestPerformStaticValidation(t *testing.T) {
	templates := []config.Template{
		{FileName: "04_api_architecture_integration.md", RequiresNonEmpty: true},
		{FileName: "05_coding_standards_guidelines.md", RequiresNonEmpty: true},
	}

	t.Run("Valid non-empty Markdown", func(t *testing.T) {
		content := "# API Guide"
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("Invalid empty Markdown", func(t *testing.T) {
		content := "   "
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
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
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
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
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
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
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err == nil {
			t.Error("expected syntax error on malformed YAML block, got nil")
		}
	})

	t.Run("Valid Mermaid diagrams", func(t *testing.T) {
		content := `
# Sequence
` + "```mermaid" + `
sequenceDiagram
  participant Alice
  participant "Audit Log" as AL
  Alice->>AL: Log event
` + "```" + `

# Gantt
` + "```mermaid" + `
gantt
  title Project Timeline
  section Setup
  API setup with database : active, m3_3, 2026-06-28, 30d
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err != nil {
			t.Errorf("expected valid Mermaid diagram to pass, got error: %v", err)
		}
	})

	t.Run("Sequence Diagram with unquoted spaces", func(t *testing.T) {
		content := `
# Invalid Sequence
` + "```mermaid" + `
sequenceDiagram
  Alice->>Audit Log: Log event
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err == nil {
			t.Error("expected syntax error for unquoted participant name with spaces, got nil")
		}
	})

	t.Run("Gantt Chart with missing colon", func(t *testing.T) {
		content := `
# Invalid Gantt
` + "```mermaid" + `
gantt
  API setup with database active, m3_3, 30d
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err == nil {
			t.Error("expected syntax error for missing colon on Gantt task line, got nil")
		}
	})

	t.Run("Unbalanced quotes in Mermaid", func(t *testing.T) {
		content := `
# Unbalanced
` + "```mermaid" + `
sequenceDiagram
  participant "Alice
` + "```" + `
`
		err := PerformStaticValidation("04_api_architecture_integration.md", content, templates)
		if err == nil {
			t.Error("expected syntax error for unbalanced quotes in Mermaid block, got nil")
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

func TestValidateMermaidBlocks_ValidSequence(t *testing.T) {
	content := "```mermaid\nsequenceDiagram\n  Alice->>Bob: Hello\n  Bob-->>Alice: Hi\n```"
	err := validateMermaidBlocks(content)
	if err != nil {
		t.Errorf("expected valid sequence to pass, got: %v", err)
	}
}

func TestValidateMermaidBlocks_ValidGantt(t *testing.T) {
	content := "```mermaid\ngantt\n  title Project\n  section S1\n  Task 1 :active, id1, 2026-01-01, 30d\n```"
	err := validateMermaidBlocks(content)
	if err != nil {
		t.Errorf("expected valid gantt to pass, got: %v", err)
	}
}

func TestValidateMermaidBlocks_UnbalancedQuotes(t *testing.T) {
	content := "```mermaid\nsequenceDiagram\n  participant \"Alice\n```"
	err := validateMermaidBlocks(content)
	if err == nil || !strings.Contains(err.Error(), "unbalanced double quotes") {
		t.Errorf("expected unbalanced quotes error, got: %v", err)
	}
}

func TestValidateMermaidBlocks_UnquotedSpaceInSequence(t *testing.T) {
	content := "```mermaid\nsequenceDiagram\n  Alice->>Bob Smith: Hello\n```"
	err := validateMermaidBlocks(content)
	if err == nil || !strings.Contains(err.Error(), "unquoted participant name with spaces") {
		t.Errorf("expected unquoted space error, got: %v", err)
	}
}

func TestValidateMermaidBlocks_MissingColonInGantt(t *testing.T) {
	content := "```mermaid\ngantt\n  Task without colon\n```"
	err := validateMermaidBlocks(content)
	if err == nil || !strings.Contains(err.Error(), "must contain a colon") {
		t.Errorf("expected missing colon error, got: %v", err)
	}
}

func TestValidateCodeBlocks_ValidJSON(t *testing.T) {
	content := "Some text\n```json\n{\"key\": \"value\"}\n```\nMore text"
	err := validateCodeBlocks(content)
	if err != nil {
		t.Errorf("expected valid JSON to pass, got: %v", err)
	}
}

func TestValidateCodeBlocks_ValidYAML(t *testing.T) {
	content := "```yaml\nkey: value\n```"
	err := validateCodeBlocks(content)
	if err != nil {
		t.Errorf("expected valid YAML to pass, got: %v", err)
	}
}

func TestValidateCodeBlocks_InvalidJSON(t *testing.T) {
	content := "```json\n{invalid json}\n```"
	err := validateCodeBlocks(content)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidateCodeBlocks_EmptyContent(t *testing.T) {
	err := validateCodeBlocks("")
	if err != nil {
		t.Errorf("expected no error for empty content, got: %v", err)
	}
}

func TestValidateJSONCodeBlock_TrailingContent(t *testing.T) {
	// Valid JSON but with trailing text (captured by regex due to fence formatting)
	code := "{\"key\": \"value\"}\n\nsome trailing text"
	err := validateJSONCodeBlock(code)
	if err != nil {
		t.Errorf("expected trailing content after JSON to be tolerated, got: %v", err)
	}
}

func TestDedent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no indentation",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "2-space common indent",
			input:    "  line1\n  line2",
			expected: "line1\nline2",
		},
		{
			name:     "mixed tab/space indent",
			input:    "\t\tline1\n\t\tline2",
			expected: "line1\nline2",
		},
		{
			name:     "already dedented",
			input:    "line1\n  indented\nline2",
			expected: "line1\n  indented\nline2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedent(tt.input)
			if got != tt.expected {
				t.Errorf("dedent(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWriteReportHeader(t *testing.T) {
	var sb strings.Builder
	writeReportHeader(&sb, "TestProj")
	result := sb.String()
	if !strings.Contains(result, "TestProj") {
		t.Errorf("expected header to contain project name")
	}
	if !strings.Contains(result, "Standards Compliance Audit Report") {
		t.Errorf("expected header to contain report title")
	}
}

func TestWriteExecutiveScorecard(t *testing.T) {
	var sb strings.Builder
	audits := []FileCompliance{
		{
			FileName: "test.md",
			Results: []gateway.ComplianceResult{
				{StandardID: "s1", Score: 100, Compliant: true},
			},
		},
	}
	standards := []config.Standard{
		{ID: "s1", Name: "Test Standard", TargetFiles: []string{"test.md"}, MinScore: 70},
	}
	writeExecutiveScorecard(&sb, audits, standards)
	result := sb.String()
	if !strings.Contains(result, "Test Standard") {
		t.Errorf("expected scorecard to contain standard name")
	}
	if !strings.Contains(result, "🟢 Compliant") {
		t.Errorf("expected scorecard to show compliant status")
	}
}

func TestWriteDetailedBreakdown(t *testing.T) {
	var sb strings.Builder
	audits := []FileCompliance{
		{
			FileName: "test.md",
			Results: []gateway.ComplianceResult{
				{StandardID: "s1", Score: 90, Compliant: true, Feedback: "Great"},
			},
		},
		{
			FileName: "broken.md",
			Err:      fmt.Errorf("static validation error"),
		},
		{
			FileName: "noresults.md",
			Results:  []gateway.ComplianceResult{},
		},
	}
	standards := []config.Standard{
		{ID: "s1", Name: "Test Standard", TargetFiles: []string{"test.md"}},
	}
	writeDetailedBreakdown(&sb, audits, standards)
	result := sb.String()
	if !strings.Contains(result, "test.md") {
		t.Errorf("expected breakdown to contain file name")
	}
	if !strings.Contains(result, "static validation error") {
		t.Errorf("expected breakdown to contain validation error")
	}
	if !strings.Contains(result, "No specific") {
		t.Errorf("expected breakdown to indicate no results for file")
	}
}

func TestWriteConsistencyCheck(t *testing.T) {
	t.Run("nil report", func(t *testing.T) {
		var sb strings.Builder
		writeConsistencyCheck(&sb, nil)
		if !strings.Contains(sb.String(), "Skipped") {
			t.Errorf("expected nil report to show 'Skipped'")
		}
	})

	t.Run("consistent", func(t *testing.T) {
		var sb strings.Builder
		writeConsistencyCheck(&sb, &gateway.ConsistencyReport{Consistent: true, Feedback: map[string]string{}})
		if !strings.Contains(sb.String(), "Passed") {
			t.Errorf("expected consistent report to show 'Passed'")
		}
	})

	t.Run("inconsistent", func(t *testing.T) {
		var sb strings.Builder
		writeConsistencyCheck(&sb, &gateway.ConsistencyReport{
			Consistent: false,
			Feedback:   map[string]string{"file.md": "mismatch detected"},
		})
		result := sb.String()
		if !strings.Contains(result, "Failed") {
			t.Errorf("expected inconsistent report to show 'Failed'")
		}
		if !strings.Contains(result, "file.md") {
			t.Errorf("expected inconsistent report to contain file name")
		}
	})
}

func TestFindResult(t *testing.T) {
	audits := []FileCompliance{
		{
			FileName: "a.md",
			Results:  []gateway.ComplianceResult{{StandardID: "s1", Score: 100}},
		},
	}
	res, found := findResult(audits, "s1")
	if !found || res.Score != 100 {
		t.Errorf("expected to find result s1 with score 100")
	}
	_, found = findResult(audits, "nonexistent")
	if found {
		t.Errorf("expected not to find nonexistent standard")
	}
}

func TestGetFailedFileError(t *testing.T) {
	audits := []FileCompliance{
		{
			FileName: "target.md",
			Err:      fmt.Errorf("error"),
		},
	}
	status, _, _, hasErr := getFailedFileError(audits, []string{"target.md"})
	if !hasErr {
		t.Errorf("expected to find error for target.md")
	}
	if !strings.Contains(status, "File Error") {
		t.Errorf("expected status to indicate file error")
	}

	// Non-matching file
	_, _, _, hasErr = getFailedFileError(audits, []string{"other.md"})
	if hasErr {
		t.Errorf("expected no error for non-matching file")
	}
}

func TestGetStandardComplianceMetrics(t *testing.T) {
	audits := []FileCompliance{
		{
			FileName: "test.md",
			Results:  []gateway.ComplianceResult{{StandardID: "s1", Score: 80, Compliant: true}},
		},
	}
	std := config.Standard{ID: "s1", MinScore: 70}

	status, score, _ := getStandardComplianceMetrics(std, audits)
	if !strings.Contains(status, "Compliant") {
		t.Errorf("expected Compliant status, got %s", status)
	}
	if !strings.Contains(score, "80") {
		t.Errorf("expected score 80%%, got %s", score)
	}
}

func TestGetStandardComplianceMetrics_NotFound(t *testing.T) {
	audits := []FileCompliance{}
	std := config.Standard{ID: "nonexistent", MinScore: 50}
	status, _, _ := getStandardComplianceMetrics(std, audits)
	if !strings.Contains(status, "Absent") {
		t.Errorf("expected Absent status for missing standard, got %s", status)
	}
}

func TestGenerateComplianceReport_WithConsistency(t *testing.T) {
	stds := []config.Standard{
		{ID: "s1", Name: "S1", TargetFiles: []string{"a.md"}, MinScore: 50},
	}
	audits := []FileCompliance{
		{
			FileName: "a.md",
			Results:  []gateway.ComplianceResult{{StandardID: "s1", Score: 100, Compliant: true}},
		},
	}
	report := GenerateComplianceReport("Proj", audits, stds, &gateway.ConsistencyReport{
		Consistent: true,
		Feedback:   map[string]string{},
	})
	if !strings.Contains(report, "Cross-Document") {
		t.Errorf("expected report to contain consistency check section")
	}
	if !strings.Contains(report, "Passed") {
		t.Errorf("expected report to show consistent")
	}
}
