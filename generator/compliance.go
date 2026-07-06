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

var codeBlockRegex = regexp.MustCompile("(?s)```(json|yaml|yml)\n(.*?)(?:\n)?```")

// PerformStaticValidation checks file syntax correctness
func PerformStaticValidation(fileName string, content string, templates []config.Template) error {
	// Find the template for this file
	var template *config.Template
	for _, t := range templates {
		if t.FileName == fileName {
			template = &t
			break
		}
	}

	// Check if template requires non-empty content
	if template != nil && template.RequiresNonEmpty {
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("generated file content is empty")
		}
	}

	if err := validateCodeBlocks(content); err != nil {
		return err
	}
	if err := validateMermaidBlocks(content); err != nil {
		return err
	}
	return nil
}

var mermaidRegex = regexp.MustCompile("(?s)```mermaid\n(.*?)(?:\n)?```")
var arrowRegex = regexp.MustCompile(`--?>>?|--?x`)

// validateMermaidBlocks checks for syntax errors in embedded Mermaid sequence diagrams and Gantt charts.
func validateMermaidBlocks(content string) error {
	matches := mermaidRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		block := match[1]

		// Check for unbalanced double quotes
		if strings.Count(block, `"`)%2 != 0 {
			return fmt.Errorf("invalid Mermaid diagram: unbalanced double quotes")
		}

		lines := strings.Split(block, "\n")
		var isSequence bool
		var isGantt bool

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
				continue
			}

			if !isSequence && !isGantt {
				if strings.HasPrefix(trimmed, "sequenceDiagram") {
					isSequence = true
					continue
				}
				if strings.HasPrefix(trimmed, "gantt") {
					isGantt = true
					continue
				}
			}

			if isSequence {
				if loc := arrowRegex.FindStringIndex(trimmed); loc != nil {
					left := strings.TrimSpace(trimmed[:loc[0]])
					rightPart := trimmed[loc[1]:]
					colonIdx := strings.Index(rightPart, ":")
					var right string
					if colonIdx != -1 {
						right = strings.TrimSpace(rightPart[:colonIdx])
					} else {
						right = strings.TrimSpace(rightPart)
					}

					if strings.Contains(left, " ") && !(strings.HasPrefix(left, `"`) && strings.HasSuffix(left, `"`)) {
						return fmt.Errorf("invalid sequence diagram: unquoted participant name with spaces: %q. Use double quotes around names with spaces", left)
					}
					if strings.Contains(right, " ") && !(strings.HasPrefix(right, `"`) && strings.HasSuffix(right, `"`)) {
						return fmt.Errorf("invalid sequence diagram: unquoted participant name with spaces: %q. Use double quotes around names with spaces", right)
					}
				}
			}

			if isGantt {
				words := strings.Fields(trimmed)
				if len(words) > 0 {
					first := words[0]
					if first != "gantt" && first != "title" && first != "dateFormat" && first != "axisFormat" && first != "section" && first != "excludes" {
						if !strings.Contains(trimmed, ":") {
							return fmt.Errorf("invalid Gantt chart: task line %q must contain a colon ':' to separate the task name and its tags/duration", trimmed)
						}
					}
				}
			}
		}
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
		code := dedent(strings.TrimSpace(match[2]))

		switch lang {
		case "json":
			if err := validateJSONCodeBlock(code); err != nil {
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

// dedent removes common leading indentation from all lines of a multiline string.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// Find the minimum common leading whitespace (spaces/tabs) across all non-empty lines
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, r := range line {
			if r == ' ' || r == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	// Strip the common prefix from each line
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}

// validateJSONCodeBlock attempts to validate JSON content, with a fallback
// for handling trailing content after the top-level JSON value (e.g., when the
// closing code fence lacks a preceding newline and the regex captures extra content).
func validateJSONCodeBlock(code string) error {
	// Primary attempt: standard strict JSON parsing
	var temp interface{}
	if err := json.Unmarshal([]byte(code), &temp); err == nil {
		return nil
	}

	// Fallback: if there's trailing content after the top-level JSON value,
	// use json.Decoder to parse just the first JSON value.
	decoder := json.NewDecoder(strings.NewReader(code))
	if err := decoder.Decode(&temp); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	// Successfully decoded the first JSON value; any remaining content is ignored.
	return nil
}
