package domain

import (
	"encoding/json"
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
	NextChoices         []string            `json:"next_choices"`
	DimensionRationales DimensionRationales `json:"dimension_rationales"`
	TokensPrompt        int                 `json:"-"`
	TokensCompletion    int                 `json:"-"`
}

// ComplianceResult represents the evaluation result of a single standard
type ComplianceResult struct {
	StandardID string `json:"standard_id"`
	Score      int    `json:"score"`
	Compliant  bool   `json:"compliant"`
	Feedback   string `json:"feedback"`
}

// UnmarshalJSON implements custom deserialization for ComplianceResult to handle raw string IDs gracefully.
func (c *ComplianceResult) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		c.StandardID = id
		c.Score = 0
		c.Compliant = false
		c.Feedback = "Auditor returned only standard ID without detailed metrics."
		return nil
	}

	type Alias ComplianceResult
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*c = ComplianceResult(aux)
	return nil
}

// Standard represents an engineering or quality standard
type Standard struct {
	ID           string   `yaml:"id" json:"id"`
	Name         string   `yaml:"name" json:"name"`
	Description  string   `yaml:"description" json:"description"`
	TargetFiles  []string `yaml:"target_files" json:"target_files"`
	Criteria     string   `yaml:"criteria" json:"criteria"`
	MinScore     int      `yaml:"min_score" json:"min_score"`
	ValidatorCmd string   `yaml:"validator_cmd,omitempty" json:"validator_cmd,omitempty"`
}

// ConsistencyReport represents the evaluation of cross-document logical consistency
type ConsistencyReport struct {
	Consistent bool              `json:"consistent"`
	Feedback   map[string]string `json:"feedback"` // fileName -> correction instructions if inconsistent
}
