package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
)

const (
	anthropicChatURL          = "https://api.anthropic.com/v1/messages"
	authApiKeyHeader          = "X-API-Key"
	anthropicVersionHeader    = "Anthropic-Version"
	anthropicVersionValue     = "2023-06-01"
	errParseAnthropicResponse = "failed to parse Anthropic response: %w"
	errEmptyContentAnthropic  = "empty content returned from Anthropic"
)

type AnthropicAdapter struct {
	apiKey string
	model  string
}

func NewAnthropicAdapter(apiKey, model string) *AnthropicAdapter {
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	return &AnthropicAdapter{
		apiKey: apiKey,
		model:  model,
	}
}

func (a *AnthropicAdapter) ProviderName() string {
	return config.ProviderAnthropic
}

func (a *AnthropicAdapter) ModelName() string {
	return a.model
}

// Anthropic API structures
type anthropicContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicMessage struct {
	Role    string                 `json:"role"` // "user" or "assistant"
	Content []anthropicContentPart `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *AnthropicAdapter) BuildOracleRequest(facts domain.Facts, history []domain.Message, latestInput string, currentScores domain.ConfidenceScores, currentRationales domain.DimensionRationales) (*http.Request, error) {
	messages := []anthropicMessage{}

	factsJSON, _ := json.Marshal(facts)
	messages = append(messages, anthropicMessage{
		Role: "user",
		Content: []anthropicContentPart{
			{Type: "text", Text: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON))},
		},
	})

	scoresJSON, _ := json.Marshal(struct {
		Scores     domain.ConfidenceScores    `json:"current_confidence_scores"`
		Rationales domain.DimensionRationales `json:"current_dimension_rationales"`
	}{
		Scores:     currentScores,
		Rationales: currentRationales,
	})
	messages = append(messages, anthropicMessage{
		Role: "user",
		Content: []anthropicContentPart{
			{Type: "text", Text: fmt.Sprintf("Current confidence scores and rationales (build upon these, do NOT reset to 0):\n%s", string(scoresJSON))},
		},
	})
	messages = append(messages, anthropicMessage{
		Role: "assistant",
		Content: []anthropicContentPart{
			{Type: "text", Text: "Acknowledged. I will cross-examine you and update these facts. Please provide your input."},
		},
	})

	for _, m := range history {
		messages = append(messages, anthropicMessage{
			Role: m.Role,
			Content: []anthropicContentPart{
				{Type: "text", Text: m.Content},
			},
		})
	}

	if latestInput != "" {
		messages = append(messages, anthropicMessage{
			Role: "user",
			Content: []anthropicContentPart{
				{Type: "text", Text: latestInput},
			},
		})
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    OracleSystemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseOracleResponse(body []byte) (*domain.OracleResponse, int, int, error) {
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseAnthropicResponse, err)
	}
	if len(anthropicResp.Content) == 0 {
		return nil, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, fmt.Errorf(errEmptyContentAnthropic)
	}

	contentStr := anthropicResp.Content[0].Text
	var oracleResp domain.OracleResponse
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		return nil, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, fmt.Errorf("invalid Oracle JSON: %w", err)
	}
	return &oracleResp, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, nil
}

func (a *AnthropicAdapter) BuildGenerateSpecRequest(facts domain.Facts, fileName string, promptTemplate string) (*http.Request, error) {
	reqBody := anthropicRequest{
		Model:  a.model,
		System: GenerateSpecSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: promptTemplate}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) BuildExtractStructuralEntitiesRequest(sourceDoc string) (*http.Request, error) {
	reqBody := anthropicRequest{
		Model:  a.model,
		System: EntityExtractionSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: sourceDoc}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseGenerateSpecResponse(body []byte) (string, int, int, error) {
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", 0, 0, fmt.Errorf(errParseAnthropicResponse, err)
	}
	if len(anthropicResp.Content) == 0 {
		return "", 0, 0, fmt.Errorf(errEmptyContentAnthropic)
	}
	return anthropicResp.Content[0].Text, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, nil
}

func (a *AnthropicAdapter) BuildEvaluateComplianceRequest(fileName string, fileContent string, standards []domain.Standard) (*http.Request, error) {
	type auditPayload struct {
		FileName    string            `json:"file_name"`
		FileContent string            `json:"file_content"`
		Standards   []domain.Standard `json:"standards"`
	}
	payloadBytes, _ := json.Marshal(auditPayload{
		FileName:    fileName,
		FileContent: fileContent,
		Standards:   standards,
	})

	reqBody := anthropicRequest{
		Model:  a.model,
		System: ComplianceSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: string(payloadBytes)}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseEvaluateComplianceResponse(body []byte) ([]domain.ComplianceResult, int, int, error) {
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseAnthropicResponse, err)
	}
	if len(anthropicResp.Content) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyContentAnthropic)
	}

	rawJSON := anthropicResp.Content[0].Text
	if idx := strings.Index(rawJSON, "{"); idx != -1 {
		if endIdx := strings.LastIndex(rawJSON, "}"); endIdx != -1 && endIdx > idx {
			rawJSON = rawJSON[idx : endIdx+1]
		}
	}

	var envelope struct {
		Results []domain.ComplianceResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		return nil, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, fmt.Errorf("invalid compliance JSON: %w", err)
	}
	return envelope.Results, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, nil
}

func (a *AnthropicAdapter) BuildRefineSpecRequest(fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (*http.Request, error) {
	var criteriaLines []string
	for _, std := range failedStandards {
		criteriaLines = append(criteriaLines, fmt.Sprintf("- Standard '%s' (%s): %s", std.Name, std.ID, std.Criteria))
	}
	criteriaText := strings.Join(criteriaLines, "\n")

	prompt := fmt.Sprintf(`We generated a specification file named "%s" but it failed standard quality checks.
Here is the feedback on why it failed:
%s

Please update the file content to address the feedback and satisfy the following standards:
%s

Reference source document:
%s

Original File Content:
%s

CRITICAL: When rewriting this file to fix the audit failures, do not abbreviate, truncate, or omit any existing sections that are already passing. You must maintain or improve the detail level of the entire document.

Return ONLY the updated file contents. Do NOT wrap it in markdown code blocks like `+"```"+` or include any conversational filler.`,
		fileName, feedback, criteriaText, strings.TrimSpace(referenceDoc), fileContent)

	reqBody := anthropicRequest{
		Model:  a.model,
		System: RefineSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: prompt}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseRefineSpecResponse(body []byte) (string, int, int, error) {
	return a.ParseGenerateSpecResponse(body)
}

func (a *AnthropicAdapter) BuildVerifyConsistencyRequest(files map[string]string) (*http.Request, error) {
	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}
	payloadBytes, _ := json.Marshal(consistencyPayload{Files: files})

	reqBody := anthropicRequest{
		Model:  a.model,
		System: ConsistencySystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: string(payloadBytes)}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseVerifyConsistencyResponse(body []byte) (*domain.ConsistencyReport, int, int, error) {
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseAnthropicResponse, err)
	}
	if len(anthropicResp.Content) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyContentAnthropic)
	}

	contentStr := SanitizeJSON(anthropicResp.Content[0].Text)
	var report domain.ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, fmt.Errorf("failed to parse consistency report JSON: %w", err)
	}
	return &report, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, nil
}

func (a *AnthropicAdapter) BuildSummarizeRequest(history []domain.Message) (*http.Request, error) {
	var messages []anthropicMessage
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, anthropicMessage{
			Role:    role,
			Content: []anthropicContentPart{{Type: "text", Text: msg.Content}},
		})
	}
	messages = append(messages, anthropicMessage{
		Role:    "user",
		Content: []anthropicContentPart{{Type: "text", Text: "Summarize the above conversation into a single paragraph capturing the key decisions and requirements."}},
	})

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    "You are a technical summarizer. Compress the conversation history into a single clear paragraph summarizing the key architectural decisions, user preferences, and engineering requirements established. Focus on consensus outcomes, not the back-and-forth dialogue.",
		Messages:  messages,
		MaxTokens: 1000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}

func (a *AnthropicAdapter) ParseSummarizeResponse(body []byte) (string, int, int, error) {
	return a.ParseGenerateSpecResponse(body)
}

func (a *AnthropicAdapter) BuildOptimizePromptRequest(files map[string]string) (*http.Request, error) {
	var sb strings.Builder
	for name, content := range files {
		sb.WriteString(fmt.Sprintf("=== FILE: %s ===\n%s\n\n", name, content))
	}

	reqBody := anthropicRequest{
		Model:  a.model,
		System: OptimizePromptSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: []anthropicContentPart{{Type: "text", Text: sb.String()}}},
		},
		MaxTokens: 4000,
	}
	return buildJSONRequest(anthropicChatURL, reqBody, map[string]string{
		authApiKeyHeader:       a.apiKey,
		anthropicVersionHeader: anthropicVersionValue,
	})
}
