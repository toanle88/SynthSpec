package gateway

import "context"

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
