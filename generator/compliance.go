package generator

import (
	"fmt"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

// FileCompliance represents the results of auditing a single generated file
type FileCompliance struct {
	FileName string
	Results  []gateway.ComplianceResult
	Err      error // Static validation or LLM call error
}

// GenerateComplianceReport compiles all results into a markdown format audit document
func GenerateComplianceReport(projectName string, fileAudits []FileCompliance, standards []config.Standard) string {
	var sb strings.Builder

	sb.WriteString("# 📋 Standards Compliance Audit Report\n\n")
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", projectName))
	sb.WriteString(fmt.Sprintf("- **Timestamp**: %s\n\n", time.Now().Format(time.RFC1123)))

	// Let's create an overview table of all 20 standards
	sb.WriteString("## Executive Scorecard\n\n")
	sb.WriteString("| Standard | Target File | Status | Score | Compliance |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	// Helper to find a result by standard ID
	findResult := func(id string) (gateway.ComplianceResult, bool) {
		for _, fa := range fileAudits {
			for _, res := range fa.Results {
				if res.StandardID == id {
					return res, true
				}
			}
		}
		return gateway.ComplianceResult{}, false
	}

	for _, std := range standards {
		fileLabel := strings.Join(std.TargetFiles, ", ")
		res, found := findResult(std.ID)

		status := "🔴 Absent"
		scoreStr := "0%"
		complianceBar := "🔴 0%"

		if found {
			scoreStr = fmt.Sprintf("%d%%", res.Score)
			if res.Compliant {
				status = "🟢 Compliant"
				complianceBar = fmt.Sprintf("🟢 %d%%", res.Score)
			} else if res.Score > 0 {
				status = "🟡 Partial"
				complianceBar = fmt.Sprintf("🟡 %d%%", res.Score)
			} else {
				status = "🔴 Non-Compliant"
				complianceBar = fmt.Sprintf("🔴 %d%%", res.Score)
			}
		} else {
			// If not targeted, check if it's because the target file failed syntax checking
			for _, fa := range fileAudits {
				for _, tf := range std.TargetFiles {
					if tf == fa.FileName && fa.Err != nil {
						status = "❌ File Error"
						scoreStr = "N/A"
						complianceBar = "❌ Error"
					}
				}
			}
		}

		sb.WriteString(fmt.Sprintf("| **%s** | %s | %s | %s | %s |\n", std.Name, fileLabel, status, scoreStr, complianceBar))
	}
	sb.WriteString("\n---\n\n")

	// Detailed breakdown per file
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
			// Get standard definition
			var stdDef config.Standard
			for _, std := range standards {
				if std.ID == res.StandardID {
					stdDef = std
					break
				}
			}

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

	return sb.String()
}

// PerformStaticValidation checks file syntax correctness
func PerformStaticValidation(fileName string, content string) error {
	switch fileName {
	case "04_api_architecture_integration.md", "05_coding_standards_guidelines.md":
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("generated file content is empty")
		}
	}
	return nil
}
