package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportToHTML(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "synthspec_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir := filepath.Join(tempDir, "output")
	distDir := filepath.Join(tempDir, "dist")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// 1. Create a dummy markdown file
	mockMD := `# Bounded Contexts
This is a test document about bounded contexts.`
	mdPath := filepath.Join(outputDir, "01_domain_model.md")
	if err := os.WriteFile(mdPath, []byte(mockMD), 0644); err != nil {
		t.Fatalf("failed to write test md: %v", err)
	}

	// 2. Create a dummy metadata file
	mockMeta := `{
		"project_name": "TestProj",
		"generation_timestamp": "2026-06-28T12:00:00Z",
		"engine_version": "1.0.0",
		"provider_used": "mock",
		"completion_metrics": {
			"total_turns": 4,
			"tokens_consumed": 1500
		},
		"compliance_summary": {
			"security": 95,
			"compliance": 88
		}
	}`
	metaPath := filepath.Join(outputDir, ".synthspec-meta.json")
	if err := os.WriteFile(metaPath, []byte(mockMeta), 0644); err != nil {
		t.Fatalf("failed to write test meta: %v", err)
	}

	// 3. Perform export
	indexPath, err := ExportToHTML("TestProj", outputDir, distDir)
	if err != nil {
		t.Fatalf("ExportToHTML failed: %v", err)
	}

	// 4. Verify outputs
	if indexPath != filepath.Join(distDir, "index.html") {
		t.Errorf("expected index path %q, got %q", filepath.Join(distDir, "index.html"), indexPath)
	}

	htmlContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}

	htmlStr := string(htmlContent)
	if !strings.Contains(htmlStr, "TestProj") {
		t.Errorf("expected HTML to contain project name 'TestProj'")
	}
	if !strings.Contains(htmlStr, "01_domain_model.md") {
		t.Errorf("expected HTML to contain filename '01_domain_model.md'")
	}
	if !strings.Contains(htmlStr, "Bounded Contexts") {
		t.Errorf("expected HTML to contain title 'Bounded Contexts'")
	}
	if !strings.Contains(htmlStr, "This is a test document about bounded contexts.") {
		t.Errorf("expected HTML to contain content")
	}
}
