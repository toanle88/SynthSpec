package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
)

func TestMockGatewayInterrogation(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()

	// Initial turn
	res, err := gw.QueryOracle(ctx, Facts{}, nil, "")
	if err != nil {
		t.Fatalf("failed to query oracle: %v", err)
	}

	if res.ConfidenceScores.Functional != 25 {
		t.Errorf("expected initial functional score to be 25, got %d", res.ConfidenceScores.Functional)
	}
	if res.NextQuestion == "" {
		t.Error("expected mock gateway to return a question")
	}

	// Complete turn (6 entries in history represents 3 full loops)
	history := []Message{
		{Role: "user", Content: "roles"},
		{Role: "assistant", Content: "question 1"},
		{Role: "user", Content: "storage"},
		{Role: "assistant", Content: "question 2"},
		{Role: "user", Content: "security"},
		{Role: "assistant", Content: "question 3"},
	}
	res2, err := gw.QueryOracle(ctx, Facts{}, history, "compliance")
	if err != nil {
		t.Fatalf("failed to query oracle: %v", err)
	}

	if res2.ConfidenceScores.Functional != 100 || res2.ConfidenceScores.Structural != 100 {
		t.Errorf("expected completed scores to be 100, got: %+v", res2.ConfidenceScores)
	}
	if res2.NextQuestion != "" {
		t.Errorf("expected next question to be empty on 100%% completion, got %q", res2.NextQuestion)
	}
}

func TestMockGateway_GenerateSpecFile_KnownFiles(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	facts := Facts{Functional: "fn", Structural: "str", Security: "sec", Compliance: "comp"}

	knownFiles := []string{
		"01_domain_model_use_cases.md",
		"02_prd_functional.md",
		"03_system_architecture.md",
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
		"06_security_threat_model.md",
		"07_engineering_roadmap.md",
	}
	for _, f := range knownFiles {
		content, err := gw.GenerateSpecFile(ctx, facts, f, "")
		if err != nil {
			t.Errorf("expected GenerateSpecFile(%q) to succeed, got: %v", f, err)
		}
		if content == "" {
			t.Errorf("expected non-empty content for %q", f)
		}
	}
}

func TestMockGateway_GenerateSpecFile_UnknownFile(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	_, err := gw.GenerateSpecFile(ctx, Facts{}, "unknown.md", "")
	if err == nil {
		t.Fatal("expected error for unknown file")
	}
}

func TestMockGateway_EvaluateCompliance(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	standards := []config.Standard{
		{ID: "clean_architecture", Name: "Clean Architecture", TargetFiles: []string{"03_system_architecture.md"}, MinScore: 70},
		{ID: "sql_parameterization", Name: "SQL Param", TargetFiles: []string{"03_system_architecture.md"}, MinScore: 80},
		{ID: "unknown_standard", Name: "Unknown", TargetFiles: []string{"03_system_architecture.md"}, MinScore: 50},
	}

	results, err := gw.EvaluateCompliance(ctx, "03_system_architecture.md", "some content", standards)
	if err != nil {
		t.Fatalf("EvaluateCompliance failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	// clean_architecture should be partial (score 70)
	if results[0].StandardID == "clean_architecture" && !results[0].Compliant {
		t.Errorf("expected clean_architecture to be compliant (score 70 >= min 70)")
	}
	// unknown_standard should be 0
	if results[2].StandardID == "unknown_standard" && results[2].Score != 0 {
		t.Errorf("expected unknown standard to have score 0, got %d", results[2].Score)
	}
}

func TestMockGateway_EvaluateCompliance_SkipsNonTargetFiles(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	standards := []config.Standard{
		{ID: "clean_architecture", TargetFiles: []string{"other.md"}},
	}
	results, err := gw.EvaluateCompliance(ctx, "03_system_architecture.md", "content", standards)
	if err != nil {
		t.Fatalf("EvaluateCompliance failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-targeting standards, got %d", len(results))
	}
}

func TestMockGateway_EvaluateCompliance_SelfCorrection(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	standards := []config.Standard{
		{ID: "unknown_standard", Name: "Unknown", TargetFiles: []string{"file.md"}, MinScore: 80},
	}
	// Content with "Fix:" triggers self-correction bump to 100
	results, err := gw.EvaluateCompliance(ctx, "file.md", "Some content with Fix: correction", standards)
	if err != nil {
		t.Fatalf("EvaluateCompliance failed: %v", err)
	}
	if len(results) != 1 || !results[0].Compliant || results[0].Score != 100 {
		t.Errorf("expected self-corrected standard to be 100%% compliant, got score=%d compliant=%v", results[0].Score, results[0].Compliant)
	}
}

func TestMockGateway_VerifyConsistency_Consistent(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	files := map[string]string{"file1.md": "content1", "file2.md": "content2"}
	report, err := gw.VerifyConsistency(ctx, files)
	if err != nil {
		t.Fatalf("VerifyConsistency failed: %v", err)
	}
	if !report.Consistent {
		t.Errorf("expected consistent=true by default")
	}
}

func TestMockGateway_VerifyConsistency_Inconsistent(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	files := map[string]string{"file1.md": "TRIGGER_INCONSISTENCY in content"}
	report, err := gw.VerifyConsistency(ctx, files)
	if err != nil {
		t.Fatalf("VerifyConsistency failed: %v", err)
	}
	if report.Consistent {
		t.Errorf("expected consistent=false when TRIGGER_INCONSISTENCY present")
	}
	if _, ok := report.Feedback["file1.md"]; !ok {
		t.Errorf("expected feedback for file1.md")
	}
}

func TestMockGateway_RefineSpecFile(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()

	content, err := gw.RefineSpecFile(ctx, "file.md", "original content", "fix it", []config.Standard{{ID: "s1"}}, "")
	if err != nil {
		t.Fatalf("RefineSpecFile failed: %v", err)
	}
	if !strings.Contains(content, "refined") || !strings.Contains(content, "s1") {
		t.Errorf("expected refined content mentioning s1, got: %s", content)
	}
}

func TestMockGateway_RefineSpecFile_WithReferenceDoc(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()

	content, err := gw.RefineSpecFile(ctx, "file.md", "original", "fix it", []config.Standard{{ID: "s1"}}, "reference doc here")
	if err != nil {
		t.Fatalf("RefineSpecFile failed: %v", err)
	}
	if !strings.Contains(content, "Reference source document preserved") {
		t.Errorf("expected reference doc preservation comment, got: %s", content)
	}
}

func TestMockGateway_QueryOracleStream(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	tokenChan := make(chan string, 100)

	res, err := gw.QueryOracleStream(ctx, Facts{}, nil, "", tokenChan)
	if err != nil {
		t.Fatalf("QueryOracleStream failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil response")
	}

	var sb strings.Builder
	for chunk := range tokenChan {
		sb.WriteString(chunk)
	}
	if sb.Len() == 0 {
		t.Error("expected non-empty stream output")
	}
}

func TestMockGateway_QueryOracleStream_WithLatestInput(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()
	tokenChan := make(chan string, 100)

	res, err := gw.QueryOracleStream(ctx, Facts{}, nil, "payment system", tokenChan)
	if err != nil {
		t.Fatalf("QueryOracleStream failed: %v", err)
	}
	if !strings.Contains(res.Facts.Functional, "payment system") {
		t.Errorf("expected facts to contain 'payment system', got: %s", res.Facts.Functional)
	}
	// Drain the channel
	for range tokenChan {
	}
}
