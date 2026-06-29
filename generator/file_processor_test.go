package generator

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func TestBuildGenerationPrompt(t *testing.T) {
	facts := gateway.Facts{
		Functional: "Build a payment system",
		Structural: "Microservices with PostgreSQL",
	}

	prompt, err := buildGenerationPrompt("Generate the {{type}} document", facts, "")
	if err != nil {
		t.Fatalf("buildGenerationPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "Generate the {{type}} document") {
		t.Errorf("expected prompt template in output")
	}
	if !strings.Contains(prompt, "Build a payment system") {
		t.Errorf("expected facts in output")
	}
}

func TestBuildGenerationPrompt_WithReference(t *testing.T) {
	facts := gateway.Facts{
		Functional: "Build a payment system",
	}
	prompt, err := buildGenerationPrompt("Generate document", facts, "# Reference Doc\nContent here")
	if err != nil {
		t.Fatalf("buildGenerationPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "Reference source document") {
		t.Errorf("expected reference doc marker in output")
	}
	if !strings.Contains(prompt, "Content here") {
		t.Errorf("expected reference content in output")
	}
}

func TestFilterApplicableStandards(t *testing.T) {
	standards := []config.Standard{
		{ID: "s1", TargetFiles: []string{"01_domain_model_use_cases.md"}},
		{ID: "s2", TargetFiles: []string{"02_prd_functional.md"}},
	}
	result := config.FilterApplicableStandards(standards, "01_domain_model_use_cases.md")
	if len(result) != 1 || result[0].ID != "s1" {
		t.Errorf("expected 1 standard (s1), got %v", result)
	}
}

func TestFilterApplicableStandards_NoMatch(t *testing.T) {
	result := config.FilterApplicableStandards(nil, "any.md")
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestBuildGenerationPromptIncludesReferenceDocument(t *testing.T) {
	facts := gateway.Facts{
		Functional: "Functional facts",
		Structural: "Structural facts",
	}

	prompt, err := buildGenerationPrompt("Write the file.\n\nUse these facts:", facts, "Domain model reference")
	if err != nil {
		t.Fatalf("failed to build generation prompt: %v", err)
	}

	if !strings.Contains(prompt, "\"functional\": \"Functional facts\"") {
		t.Fatalf("expected prompt to include serialized facts, got: %s", prompt)
	}

	if !strings.Contains(prompt, "Reference source document:") {
		t.Fatalf("expected prompt to include reference document marker, got: %s", prompt)
	}

	if !strings.Contains(prompt, "Domain model reference") {
		t.Fatalf("expected prompt to include reference document content, got: %s", prompt)
	}

	if strings.Index(prompt, "Reference source document:") < strings.Index(prompt, "\"functional\": \"Functional facts\"") {
		t.Fatalf("expected reference document to appear after facts, got: %s", prompt)
	}
}
