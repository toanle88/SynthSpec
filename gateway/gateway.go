package gateway

import (
	"context"
	"strings"
)

// Message represents a single turn in the conversation history
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// Facts represents the compiled specification details
type Facts struct {
	Functional string `json:"functional"`
	Structural string `json:"structural"`
	Security   string `json:"security"`
	Compliance string `json:"compliance"`
}

// ConfidenceScores represents confidence levels (0-100) across 4 dimensions
type ConfidenceScores struct {
	Functional int `json:"functional"`
	Structural int `json:"structural"`
	Security   int `json:"security"`
	Compliance int `json:"compliance"`
}

// DimensionRationales contains brief reasoning behind each dimension's score
type DimensionRationales struct {
	Functional string `json:"functional"`
	Structural string `json:"structural"`
	Security   string `json:"security"`
	Compliance string `json:"compliance"`
}

// OracleResponse represents the structured JSON response from the LLM Oracle
type OracleResponse struct {
	Facts               Facts               `json:"facts"`
	ConfidenceScores    ConfidenceScores    `json:"confidence_scores"`
	NextQuestion        string              `json:"next_question"`
	DimensionRationales DimensionRationales `json:"dimension_rationales"`
	TokensPrompt        int                 `json:"-"`
	TokensCompletion    int                 `json:"-"`
}

// Gateway defines the uniform interface for communicating with upstream LLMs
type Gateway interface {
	// QueryOracle sends the current facts, conversation history, and latest input
	// to get the next interrogation state from the Oracle.
	QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error)

	// GenerateSpecFile generates the contents of a specific output asset based on the compiled facts.
	GenerateSpecFile(ctx context.Context, facts Facts, fileName string) (string, error)
}

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
