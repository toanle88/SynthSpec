package generator

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func TestCollectFailedStandards(t *testing.T) {
	standards := []config.Standard{
		{ID: "s1", Name: "Standard 1", MinScore: 80},
		{ID: "s2", Name: "Standard 2", MinScore: 60},
	}
	results := []gateway.ComplianceResult{
		{StandardID: "s1", Score: 50, Compliant: false},
		{StandardID: "s2", Score: 100, Compliant: true},
	}
	failed, _ := collectFailedStandards(results, standards)
	if len(failed) != 1 || failed[0].ID != "s1" {
		t.Errorf("expected 1 failed standard (s1), got %d", len(failed))
	}
}

func TestUpdateComplianceResultWithValidationError(t *testing.T) {
	res := &gateway.ComplianceResult{Score: 100, Compliant: true, Feedback: "All good"}
	updateComplianceResultWithValidationError(res, fmt.Errorf("validator failed"), "output from validator")
	if res.Compliant {
		t.Errorf("expected compliant=false after validation error")
	}
	if res.Score != 0 {
		t.Errorf("expected score 0 after validation error, got %d", res.Score)
	}
}

func TestGetOrInsertResult(t *testing.T) {
	results := []gateway.ComplianceResult{}
	resultsMap := make(map[string]*gateway.ComplianceResult)

	// Insert new
	r := getOrInsertResult("s1", resultsMap, &results)
	if r.StandardID != "s1" || r.Score != 100 {
		t.Errorf("expected new result with default score 100")
	}

	// Get existing
	r2 := getOrInsertResult("s1", resultsMap, &results)
	if r2 != r {
		t.Errorf("expected same pointer for existing result")
	}
}

func TestRunExternalValidator(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-val-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	ctx := context.Background()

	// Test success command
	successCmd := "echo success"

	out, err := runExternalValidator(ctx, successCmd, tempFile.Name())
	if err != nil {
		t.Errorf("expected no error, got: %v, output: %q", err, out)
	}
	if !strings.Contains(out, "success") {
		t.Errorf("expected output to contain 'success', got %q", out)
	}

	// Test failing command
	var failCmd string
	if runtime.GOOS == "windows" {
		failCmd = "type non_existent_file_12345.txt"
	} else {
		failCmd = "cat non_existent_file_12345.txt"
	}

	_, err = runExternalValidator(ctx, failCmd, tempFile.Name())
	if err == nil {
		t.Error("expected error for failing command, got nil")
	}
}
