package generator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/toanle/synthspec/config"
	"gopkg.in/yaml.v3"
)

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
	if err := LintMarkdown(content); err != nil {
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
		if err := processMermaidBlock(match[1]); err != nil {
			return err
		}
	}
	return nil
}

func processMermaidBlock(block string) error {
	if strings.Count(block, `"`)%2 != 0 {
		return fmt.Errorf("invalid Mermaid diagram: unbalanced double quotes")
	}

	lines := strings.Split(block, "\n")
	var isSequence, isGantt bool

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
			if err := validateSequenceLine(trimmed); err != nil {
				return err
			}
		}

		if isGantt {
			if err := validateGanttLine(trimmed); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSequenceLine(trimmed string) error {
	loc := arrowRegex.FindStringIndex(trimmed)
	if loc == nil {
		return nil
	}
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
	return nil
}

func validateGanttLine(trimmed string) error {
	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return nil
	}
	first := words[0]
	if first != "gantt" && first != "title" && first != "dateFormat" && first != "axisFormat" && first != "section" && first != "excludes" {
		if !strings.Contains(trimmed, ":") {
			return fmt.Errorf("invalid Gantt chart: task line %q must contain a colon ':' to separate the task name and its tags/duration", trimmed)
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

	minIndent := getMinIndent(lines)
	if minIndent <= 0 {
		return s
	}

	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}

func getMinIndent(lines []string) int {
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
	return minIndent
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
