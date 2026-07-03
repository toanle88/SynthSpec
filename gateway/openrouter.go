package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/shared"
)

const (
	openrouterChatURL          = "https://openrouter.ai/api/v1/chat/completions"
	errParseOpenRouterResponse = "failed to parse OpenRouter chat response: %w"
	errEmptyChoiceOpenRouter   = "empty choice array returned from OpenRouter"

	refererHeader = "HTTP-Referer"
	refererValue  = "https://github.com/toanle/synthspec"
	xTitleHeader  = "X-Title"
)

type OpenRouterGateway struct {
	apiKey     string
	model      string
	client     *http.Client
	maxRetries int
}

func NewOpenRouterGateway(apiKey, model string) *OpenRouterGateway {
	if model == "" {
		model = "meta-llama/llama-3.1-405b-instruct"
	}
	timeout := 5 * time.Minute
	maxRetries := 3

	if s, err := config.LoadSettings(); err == nil && s != nil {
		timeout = time.Duration(s.TimeoutSeconds) * time.Second
		maxRetries = s.MaxRetries
	}

	return &OpenRouterGateway{
		apiKey:     apiKey,
		model:      model,
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
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

func (o *OpenRouterGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
	systemPrompt := OracleSystemPrompt

	messages := []openRouterChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// Add facts context to boot the history
	factsJSON, _ := json.Marshal(facts)
	messages = append(messages, openRouterChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON)),
	})

	// Add conversation history
	for _, m := range history {
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		messages = append(messages, openRouterChatMessage{Role: role, Content: m.Content})
	}

	// Add latest user input
	if latestInput != "" {
		messages = append(messages, openRouterChatMessage{Role: "user", Content: latestInput})
	}

	reqBody := openRouterChatRequest{
		Model:          o.model,
		Messages:       messages,
		ResponseFormat: &openRouterResponseFormat{Type: "json_object"},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	startTime := time.Now()
	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	duration := time.Since(startTime)
	if err != nil {
		logger.LogAPI(config.ProviderOpenRouter, o.model, duration, 0, 0, err)
		return nil, err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		logger.LogAPI(config.ProviderOpenRouter, o.model, duration, 0, 0, err)
		return nil, fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		errEmpty := fmt.Errorf(errEmptyChoiceOpenRouter)
		logger.LogAPI(config.ProviderOpenRouter, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errEmpty)
		return nil, errEmpty
	}

	var oracleResp OracleResponse
	contentStr := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if contentStr == "" {
		errEmpty := fmt.Errorf("LLM returned an empty response. This can happen with reasoning models or transient provider errors on OpenRouter. Please try submitting again.")
		logger.LogAPI(config.ProviderOpenRouter, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errEmpty)
		return nil, errEmpty
	}
	contentStr = shared.SanitizeJSON(contentStr)
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		errInvalidJSON := fmt.Errorf("LLM returned invalid Oracle JSON: %w (Raw content: %s)", err, contentStr)
		logger.LogAPI(config.ProviderOpenRouter, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errInvalidJSON)
		return nil, errInvalidJSON
	}

	logger.LogAPI(config.ProviderOpenRouter, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil)

	oracleResp.TokensPrompt = chatResp.Usage.PromptTokens
	oracleResp.TokensCompletion = chatResp.Usage.CompletionTokens
	oracleResp.NextQuestion = shared.SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (o *OpenRouterGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := o.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	shared.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (o *OpenRouterGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	messages := []openRouterChatMessage{
		{Role: "system", Content: GenerateSpecSystemPrompt},
		{Role: "user", Content: promptTemplate},
	}

	reqBody := openRouterChatRequest{
		Model:    o.model,
		Messages: messages,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return "", err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (o *OpenRouterGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
	var applicableStandards []domain.Standard
	for _, std := range standards {
		for _, tf := range std.TargetFiles {
			if tf == fileName {
				applicableStandards = append(applicableStandards, std)
				break
			}
		}
	}

	if len(applicableStandards) == 0 {
		return nil, nil
	}

	systemPrompt := ComplianceSystemPrompt

	type auditPayload struct {
		FileName    string            `json:"file_name"`
		FileContent string            `json:"file_content"`
		Standards   []domain.Standard `json:"standards"`
	}

	payloadStruct := auditPayload{
		FileName:    fileName,
		FileContent: fileContent,
		Standards:   applicableStandards,
	}
	payloadBytes, _ := json.Marshal(payloadStruct)

	messages := []openRouterChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(payloadBytes)},
	}

	reqBody := openRouterChatRequest{
		Model:          o.model,
		Messages:       messages,
		ResponseFormat: &openRouterResponseFormat{Type: "json_object"},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return nil, err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	rawJSON := chatResp.Choices[0].Message.Content

	var envelope struct {
		Results []ComplianceResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		return nil, fmt.Errorf("OpenRouter returned invalid compliance JSON: %w (Raw content: %s)", err, rawJSON)
	}

	return envelope.Results, nil
}

func (o *OpenRouterGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
	systemPrompt := "You are a senior solutions architect. Your job is to modify an existing specification file to fix quality standards violations. Return only the updated file contents and nothing else. No preamble, no postamble, no markdown codeblocks unless specified."

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
%sRefineSystemPrompt

CRITICAL: When rewriting this file to fix the audit failures, do not abbreviate, truncate, or omit any existing sections that are already passing. You must maintain or improve the detail level of the entire document.

Return ONLY the updated file contents. Do NOT wrap it in markdown code blocks like `+"```"+` or include any conversational filler.`,
		fileName, feedback, criteriaText, strings.TrimSpace(referenceDoc), fileContent)

	messages := []openRouterChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	reqBody := openRouterChatRequest{
		Model:    o.model,
		Messages: messages,
	}

	if fileName == "05_engineering_backlog.json" {
		reqBody.ResponseFormat = &openRouterResponseFormat{Type: "json_object"}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return "", err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (o *OpenRouterGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	systemPrompt := ConsistencySystemPrompt

	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}

	payloadStruct := consistencyPayload{Files: files}
	payloadBytes, _ := json.Marshal(payloadStruct)

	messages := []openRouterChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(payloadBytes)},
	}

	reqBody := openRouterChatRequest{
		Model:    o.model,
		Messages: messages,
		ResponseFormat: &openRouterResponseFormat{
			Type: "json_object",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return nil, err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	contentStr := chatResp.Choices[0].Message.Content
	contentStr = shared.SanitizeJSON(contentStr)

	var report ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse consistency report JSON: %w (Raw content: %s)", err, contentStr)
	}

	return &report, nil
}

// Summarize generates a concise summary of the conversation history using the OpenRouter model.
func (o *OpenRouterGateway) Summarize(ctx context.Context, history []Message) (string, error) {
	systemPrompt := "You are a technical summarizer. Compress the conversation history into a single clear paragraph summarizing the key architectural decisions, user preferences, and engineering requirements established. Focus on consensus outcomes, not the back-and-forth dialogue."

	messages := []openRouterChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// Add conversation history
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, openRouterChatMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add summarization instruction
	messages = append(messages, openRouterChatMessage{
		Role:    "user",
		Content: "Summarize the above conversation into a single paragraph capturing the key decisions and requirements.",
	})

	reqBody := openRouterChatRequest{
		Model:    o.model,
		Messages: messages,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openrouterChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)
	req.Header.Set(refererHeader, refererValue)
	req.Header.Set(xTitleHeader, "SynthSpec")

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return "", err
	}

	var chatResp openRouterChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf(errParseOpenRouterResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf(errEmptyChoiceOpenRouter)
	}

	return chatResp.Choices[0].Message.Content, nil
}
