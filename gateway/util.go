package gateway

import (
	"strings"
)

// SanitizeNextQuestion enforces the strict single question constraint on LLM output.
// It truncates the output up to the first question mark (if present) and cleans list markers.
func SanitizeNextQuestion(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}

	// 1. If it starts with common list markers like "- ", "* ", "1. ", remove them
	prefixes := []string{"-", "*", "•", "1.", "2.", "3."}
	for {
		cleaned := false
		for _, pref := range prefixes {
			trimmed := strings.TrimSpace(q)
			if strings.HasPrefix(trimmed, pref) {
				q = strings.TrimPrefix(trimmed, pref)
				cleaned = true
			}
		}
		if !cleaned {
			break
		}
	}
	q = strings.TrimSpace(q)

	// 2. Truncate at the first question mark if it exists to enforce strict single question
	if idx := strings.Index(q, "?"); idx != -1 {
		return q[:idx+1]
	}

	// 3. Otherwise, split by newline and take the first non-empty line
	lines := strings.Split(q, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}

	return q
}

// SanitizeJSON strips markdown code block fences if they exist
func SanitizeJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	return content
}
