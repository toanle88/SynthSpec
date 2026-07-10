package generator

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
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
