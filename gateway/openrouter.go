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
	openrouterChatURL          = "https://openrouter.ai/api/v1/chat/completions"
	errParseOpenRouterResponse = "failed to parse OpenRouter chat response: %w"
	errEmptyChoiceOpenRouter   = "empty choice array returned from OpenRouter"

	refererHeader = "HTTP-Referer"
	refererValue  = "https://github.com/toanle/synthspec"
	xTitleHeader  = "X-Title"
)

type OpenRouterAdapter struct {
	apiKey string
	model  string
}

func NewOpenRouterAdapter(apiKey, model string) *OpenRouterAdapter {
	if model == "" {
		model = "meta-llama/llama-3.1-405b-instruct"
	}
	return &OpenRouterAdapter{
		apiKey: apiKey,
		model:  model,
	}
}

func (o *OpenRouterAdapter) ProviderName() string {
	return config.ProviderOpenRouter
}

func (o *OpenRouterAdapter) ModelName() string {
	return o.model
}

// OpenRouter API JSON Schemas
type openRouterChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterChatRequest struct {
	Model          string                    `json:"model"`
	Messages       []openRouterChatMessage   `json:"messages"`
	ResponseFormat *openRouterResponseFormat `json:"response_format,omitempty"`
}

type openRouterResponseFormat struct {
	Type string `json:"type"`
}

type openRouterChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (o *OpenRouterAdapter) BuildOracleRequest(facts domain.Facts, history []domain.Message, latestInput string, currentScores domain.ConfidenceScores, currentRationales domain.DimensionRationales) (*http.Request, error) {
	messages := []openRouterChatMessage{
		{Role: "system", Content: OracleSystemPrompt},
	}

	factsJSON, _ := json.Marshal(facts)
	messages = append(messages, openRouterChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON)),
	})

	scoresJSON, _ := json.Marshal(struct {
		Scores     domain.ConfidenceScores    `json:"current_confidence_scores"`
		Rationales domain.DimensionRationales `json:"current_dimension_rationales"`
	}{
		Scores:     currentScores,
		Rationales: currentRationales,
	})
	messages = append(messages, openRouterChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("Current confidence scores and rationales (build upon these, do NOT reset to 0):\n%s", string(scoresJSON)),
	})

	for _, m := range history {
		role := m.Role
		if role != "assistant" {
			role = "user"
		}
		messages = append(messages, openRouterChatMessage{Role: role, Content: m.Content})
	}

	if latestInput != "" {
		messages = append(messages, openRouterChatMessage{Role: "user", Content: latestInput})
	}

	reqBody := openRouterChatRequest{
		Model:          o.model,
		Messages:       messages,
		ResponseFormat: &openRouterResponseFormat{Type: "json_object"},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseOracleResponse(body []byte) (*domain.OracleResponse, int, int, error) {
	var chatResp openRouterChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	contentStr := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if contentStr == "" {
		return nil, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, fmt.Errorf("LLM returned an empty response. This can happen with reasoning models or transient provider errors on OpenRouter. Please try submitting again.")
	}
	contentStr = SanitizeJSON(contentStr)
	var oracleResp domain.OracleResponse
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		return nil, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, fmt.Errorf("invalid Oracle JSON: %w", err)
	}
	return &oracleResp, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil
}

func (o *OpenRouterAdapter) BuildGenerateSpecRequest(facts domain.Facts, fileName string, promptTemplate string) (*http.Request, error) {
	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: GenerateSpecSystemPrompt},
			{Role: "user", Content: promptTemplate},
		},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) BuildExtractStructuralEntitiesRequest(sourceDoc string) (*http.Request, error) {
	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: EntityExtractionSystemPrompt},
			{Role: "user", Content: sourceDoc},
		},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseGenerateSpecResponse(body []byte) (string, int, int, error) {
	var chatResp openRouterChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", 0, 0, fmt.Errorf(errParseOpenRouterResponse, err)
	}
	if len(chatResp.Choices) == 0 {
		return "", 0, 0, fmt.Errorf(errEmptyChoiceOpenRouter)
	}
	return chatResp.Choices[0].Message.Content, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil
}

func (o *OpenRouterAdapter) BuildEvaluateComplianceRequest(fileName string, fileContent string, standards []domain.Standard) (*http.Request, error) {
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

	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: ComplianceSystemPrompt},
			{Role: "user", Content: string(payloadBytes)},
		},
		ResponseFormat: &openRouterResponseFormat{Type: "json_object"},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseEvaluateComplianceResponse(body []byte) ([]domain.ComplianceResult, int, int, error) {
	var chatResp openRouterChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseOpenRouterResponse, err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	rawJSON := chatResp.Choices[0].Message.Content
	var envelope struct {
		Results []domain.ComplianceResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		return nil, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, fmt.Errorf("OpenRouter returned invalid compliance JSON: %w", err)
	}
	return envelope.Results, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil
}

func (o *OpenRouterAdapter) BuildRefineSpecRequest(fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (*http.Request, error) {
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

	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: RefineSystemPrompt},
			{Role: "user", Content: prompt},
		},
	}
	if fileName == "05_engineering_backlog.json" {
		reqBody.ResponseFormat = &openRouterResponseFormat{Type: "json_object"}
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseRefineSpecResponse(body []byte) (string, int, int, error) {
	return o.ParseGenerateSpecResponse(body)
}

func (o *OpenRouterAdapter) BuildVerifyConsistencyRequest(files map[string]string) (*http.Request, error) {
	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}
	payloadBytes, _ := json.Marshal(consistencyPayload{Files: files})

	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: ConsistencySystemPrompt},
			{Role: "user", Content: string(payloadBytes)},
		},
		ResponseFormat: &openRouterResponseFormat{Type: "json_object"},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseVerifyConsistencyResponse(body []byte) (*domain.ConsistencyReport, int, int, error) {
	var chatResp openRouterChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseOpenRouterResponse, err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	contentStr := SanitizeJSON(chatResp.Choices[0].Message.Content)
	var report domain.ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, fmt.Errorf("failed to parse consistency report JSON: %w", err)
	}
	return &report, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil
}

func (o *OpenRouterAdapter) BuildSummarizeRequest(history []domain.Message) (*http.Request, error) {
	messages := []openRouterChatMessage{
		{Role: "system", Content: "You are a technical summarizer. Compress the conversation history into a single clear paragraph summarizing the key architectural decisions, user preferences, and engineering requirements established. Focus on consensus outcomes, not the back-and-forth dialogue."},
	}
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, openRouterChatMessage{Role: role, Content: msg.Content})
	}
	messages = append(messages, openRouterChatMessage{
		Role:    "user",
		Content: "Summarize the above conversation into a single paragraph capturing the key decisions and requirements.",
	})

	reqBody := openRouterChatRequest{
		Model:    o.model,
		Messages: messages,
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}

func (o *OpenRouterAdapter) ParseSummarizeResponse(body []byte) (string, int, int, error) {
	return o.ParseGenerateSpecResponse(body)
}

func (o *OpenRouterAdapter) BuildOptimizePromptRequest(files map[string]string) (*http.Request, error) {
	var sb strings.Builder
	for name, content := range files {
		sb.WriteString(fmt.Sprintf("=== FILE: %s ===\n%s\n\n", name, content))
	}

	reqBody := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{Role: "system", Content: OptimizePromptSystemPrompt},
			{Role: "user", Content: sb.String()},
		},
	}
	return buildJSONRequest(openrouterChatURL, reqBody, map[string]string{
		"Authorization": authBearerPrefix + o.apiKey,
		refererHeader:   refererValue,
		xTitleHeader:    "SynthSpec",
	})
}
