package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

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
