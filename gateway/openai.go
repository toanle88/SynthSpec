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
	openaiChatURL          = "https://api.openai.com/v1/chat/completions"
	errParseOpenAIResponse = "failed to parse OpenAI chat response: %w"
	errEmptyChoiceOpenAI   = "empty choice array returned from OpenAI"
)

type OpenAIGateway struct {
	apiKey     string
	model      string
	client     *http.Client
	maxRetries int
}

func NewOpenAIGateway(apiKey, model string) *OpenAIGateway {
	if model == "" {
		model = "gpt-4o"
	}
	timeout := 5 * time.Minute
	maxRetries := 3

	if s, err := config.LoadSettings(); err == nil && s != nil {
		timeout = time.Duration(s.TimeoutSeconds) * time.Second
		maxRetries = s.MaxRetries
	}

	return &OpenAIGateway{
		apiKey:     apiKey,
		model:      model,
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
}

// OpenAI API JSON Schemas
type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	ResponseFormat *responseFormat     `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
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

func (o *OpenAIGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
	systemPrompt := OracleSystemPrompt

	messages := []openAIChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// Add facts context to boot the history
	factsJSON, _ := json.Marshal(facts)
	messages = append(messages, openAIChatMessage{
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
		messages = append(messages, openAIChatMessage{Role: role, Content: m.Content})
	}

	// Add latest user input
	if latestInput != "" {
		messages = append(messages, openAIChatMessage{Role: "user", Content: latestInput})
	}

	reqBody := openAIChatRequest{
		Model:          o.model,
		Messages:       messages,
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)

	startTime := time.Now()
	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	duration := time.Since(startTime)
	if err != nil {
		logger.LogAPI(config.ProviderOpenAI, o.model, duration, 0, 0, err)
		return nil, err
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		logger.LogAPI(config.ProviderOpenAI, o.model, duration, 0, 0, err)
		return nil, fmt.Errorf(errParseOpenAIResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		errEmpty := fmt.Errorf(errEmptyChoiceOpenAI)
		logger.LogAPI(config.ProviderOpenAI, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errEmpty)
		return nil, errEmpty
	}

	var oracleResp OracleResponse
	contentStr := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if contentStr == "" {
		errEmpty := fmt.Errorf("LLM returned an empty response. This can happen with reasoning models or transient provider errors. Please try submitting again.")
		logger.LogAPI(config.ProviderOpenAI, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errEmpty)
		return nil, errEmpty
	}
	contentStr = shared.SanitizeJSON(contentStr)
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		errInvalidJSON := fmt.Errorf("LLM returned invalid Oracle JSON: %w (Raw content: %s)", err, contentStr)
		logger.LogAPI(config.ProviderOpenAI, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, errInvalidJSON)
		return nil, errInvalidJSON
	}

	logger.LogAPI(config.ProviderOpenAI, o.model, duration, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil)

	oracleResp.TokensPrompt = chatResp.Usage.PromptTokens
	oracleResp.TokensCompletion = chatResp.Usage.CompletionTokens
	oracleResp.NextQuestion = shared.SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (o *OpenAIGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := o.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	shared.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (o *OpenAIGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	messages := []openAIChatMessage{
		{Role: "system", Content: GenerateSpecSystemPrompt},
		{Role: "user", Content: promptTemplate},
	}

	reqBody := openAIChatRequest{
		Model:    o.model,
		Messages: messages,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return "", err
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf(errParseOpenAIResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf(errEmptyChoiceOpenAI)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (o *OpenAIGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
	applicableStandards := config.FilterApplicableStandards(standards, fileName)

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

	messages := []openAIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(payloadBytes)},
	}

	reqBody := openAIChatRequest{
		Model:          o.model,
		Messages:       messages,
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return nil, err
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf(errParseOpenAIResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf(errEmptyChoiceOpenAI)
	}

	rawJSON := chatResp.Choices[0].Message.Content

	var envelope struct {
		Results []ComplianceResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		return nil, fmt.Errorf("OpenAI returned invalid compliance JSON: %w (Raw content: %s)", err, rawJSON)
	}

	return envelope.Results, nil
}

func (o *OpenAIGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
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

	messages := []openAIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	reqBody := openAIChatRequest{
		Model:    o.model,
		Messages: messages,
	}

	if fileName == "05_engineering_backlog.json" {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return "", err
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf(errParseOpenAIResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf(errEmptyChoiceOpenAI)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (o *OpenAIGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	systemPrompt := ConsistencySystemPrompt

	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}

	payloadStruct := consistencyPayload{Files: files}
	payloadBytes, _ := json.Marshal(payloadStruct)

	messages := []openAIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(payloadBytes)},
	}

	reqBody := openAIChatRequest{
		Model:    o.model,
		Messages: messages,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set("Authorization", authBearerPrefix+o.apiKey)

	respBytes, err := SendWithRetry(ctx, o.client, req, o.maxRetries)
	if err != nil {
		return nil, err
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf(errParseOpenAIResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf(errEmptyChoiceOpenAI)
	}

	contentStr := chatResp.Choices[0].Message.Content
	contentStr = shared.SanitizeJSON(contentStr)

	var report ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse consistency report JSON: %w (Raw content: %s)", err, contentStr)
	}

	return &report, nil
}
