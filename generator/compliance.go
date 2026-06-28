package generator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"gopkg.in/yaml.v3"
)

// FileCompliance represents the results of auditing a single generated file
type FileCompliance struct {
	FileName string
	Results  []gateway.ComplianceResult
	Err      error // Static validation or LLM call error
}

// GenerateComplianceReport compiles all results into a markdown format audit document
// GenerateComplianceReport compiles all results into a markdown format audit document
func GenerateComplianceReport(projectName string, fileAudits []FileCompliance, standards []config.Standard, consistencyReport *gateway.ConsistencyReport) string {
	var sb strings.Builder

	writeReportHeader(&sb, projectName)
	writeExecutiveScorecard(&sb, fileAudits, standards)
	writeDetailedBreakdown(&sb, fileAudits, standards)
	writeConsistencyCheck(&sb, consistencyReport)

	return sb.String()
}

func writeReportHeader(sb *strings.Builder, projectName string) {
	sb.WriteString("# 📋 Standards Compliance Audit Report\n\n")
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", projectName))
	sb.WriteString(fmt.Sprintf("- **Timestamp**: %s\n\n", time.Now().Format(time.RFC1123)))
}

func findResult(fileAudits []FileCompliance, id string) (gateway.ComplianceResult, bool) {
	for _, fa := range fileAudits {
		for _, res := range fa.Results {
			if res.StandardID == id {
				return res, true
			}
		}
	}
	return gateway.ComplianceResult{}, false
}

func getFailedFileError(fileAudits []FileCompliance, targetFiles []string) (string, string, string, bool) {
	for _, fa := range fileAudits {
		for _, tf := range targetFiles {
			if tf == fa.FileName && fa.Err != nil {
				return "❌ File Error", "N/A", "❌ Error", true
			}
		}
	}
	return "", "", "", false
}

func getStandardComplianceMetrics(std config.Standard, fileAudits []FileCompliance) (string, string, string) {
	res, found := findResult(fileAudits, std.ID)
	if !found {
		if status, scoreStr, complianceBar, hasErr := getFailedFileError(fileAudits, std.TargetFiles); hasErr {
			return status, scoreStr, complianceBar
		}
		return "🔴 Absent", "0%", "🔴 0%"
	}

	scoreStr := fmt.Sprintf("%d%%", res.Score)
	if res.Compliant {
		return "🟢 Compliant", scoreStr, fmt.Sprintf("🟢 %d%%", res.Score)
	}
	if res.Score > 0 {
		return "🟡 Partial", scoreStr, fmt.Sprintf("🟡 %d%%", res.Score)
	}
	return "🔴 Non-Compliant", scoreStr, fmt.Sprintf("🔴 %d%%", res.Score)
}

func writeExecutiveScorecard(sb *strings.Builder, fileAudits []FileCompliance, standards []config.Standard) {
	sb.WriteString("## Executive Scorecard\n\n")
	sb.WriteString("| Standard | Target File | Status | Score | Compliance |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, std := range standards {
		fileLabel := strings.Join(std.TargetFiles, ", ")
		status, scoreStr, complianceBar := getStandardComplianceMetrics(std, fileAudits)
		sb.WriteString(fmt.Sprintf("| **%s** | %s | %s | %s | %s |\n", std.Name, fileLabel, status, scoreStr, complianceBar))
	}
	sb.WriteString("\n---\n\n")
}

func findStandardDef(standards []config.Standard, standardID string) (config.Standard, bool) {
	for _, std := range standards {
		if std.ID == standardID {
			return std, true
		}
	}
	return config.Standard{}, false
}

func writeDetailedBreakdown(sb *strings.Builder, fileAudits []FileCompliance, standards []config.Standard) {
	sb.WriteString("## Detailed Audit Breakdown\n\n")
	for _, fa := range fileAudits {
		sb.WriteString(fmt.Sprintf("### 📁 %s\n\n", fa.FileName))
		if fa.Err != nil {
			sb.WriteString(fmt.Sprintf("> ❌ **Static Validation Failed**: %v\n\n", fa.Err))
			continue
		}

		if len(fa.Results) == 0 {
			sb.WriteString("No specific architectural quality standards mapped to this file.\n\n")
			continue
		}

		sb.WriteString("| Standard ID | Quality Check | Score | Status | Feedback / Remediation |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

		for _, res := range fa.Results {
			stdDef, _ := findStandardDef(standards, res.StandardID)
			status := "🔴 Non-Compliant"
			if res.Compliant {
				status = "🟢 Compliant"
			} else if res.Score > 0 {
				status = "🟡 Partial"
			}

			sb.WriteString(fmt.Sprintf("| `%s` | %s | %d%% | %s | %s |\n",
				res.StandardID, stdDef.Name, res.Score, status, res.Feedback))
		}
		sb.WriteString("\n")
	}
}

func writeConsistencyCheck(sb *strings.Builder, consistencyReport *gateway.ConsistencyReport) {
	sb.WriteString("## 🔄 Cross-Document Consistency Check\n\n")
	if consistencyReport == nil {
		sb.WriteString("⚠️ **Skipped**: No consistency report was generated.\n\n")
		return
	}

	if consistencyReport.Consistent {
		sb.WriteString("🟢 **Passed**: All generated documents are logically and structurally consistent with one another.\n")
	} else {
		sb.WriteString("🔴 **Failed**: Semantic discrepancies were detected between the generated files:\n\n")
		for file, feedback := range consistencyReport.Feedback {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", file, feedback))
		}
	}
	sb.WriteString("\n")
}

var codeBlockRegex = regexp.MustCompile("(?s)```(json|yaml|yml)\n(.*?)\n```")

// PerformStaticValidation checks file syntax correctness
func PerformStaticValidation(fileName string, content string) error {
	switch fileName {
	case "04_api_architecture_integration.md", "05_coding_standards_guidelines.md":
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("generated file content is empty")
		}
	}

	if err := validateCodeBlocks(content); err != nil {
		return err
	}
	return nil
}

// validateCodeBlocks parses and validates all yaml and json blocks in the content.
func validateCodeBlocks(content string) error {
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		lang := match[1]
		code := strings.TrimSpace(match[2])

		switch lang {
		case "json":
			var temp interface{}
			if err := json.Unmarshal([]byte(code), &temp); err != nil {
				return fmt.Errorf("invalid json code block: %w", err)
			}
		case "yaml", "yml":
			var temp interface{}
			if err := yaml.Unmarshal([]byte(code), &temp); err != nil {
				return fmt.Errorf("invalid yaml code block: %w", err)
			}
		}
	}
	return nil
}
