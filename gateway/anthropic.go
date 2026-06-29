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
	anthropicChatURL          = "https://api.anthropic.com/v1/messages"
	authApiKeyHeader          = "X-API-Key"
	anthropicVersionHeader    = "Anthropic-Version"
	anthropicVersionValue     = "2023-06-01"
	errParseAnthropicResponse = "failed to parse Anthropic response: %w"
	errEmptyContentAnthropic  = "empty content returned from Anthropic"
)

type AnthropicGateway struct {
	apiKey     string
	model      string
	client     *http.Client
	maxRetries int
}

func NewAnthropicGateway(apiKey, model string) *AnthropicGateway {
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	timeout := 5 * time.Minute
	maxRetries := 3

	if s, err := config.LoadSettings(); err == nil && s != nil {
		timeout = time.Duration(s.TimeoutSeconds) * time.Second
		maxRetries = s.MaxRetries
	}

	return &AnthropicGateway{
		apiKey:     apiKey,
		model:      model,
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
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

func (a *AnthropicGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
	systemPrompt := OracleSystemPrompt

	messages := []anthropicMessage{}

	// Add facts context to boot history (Claude does not allow system messages inside message array, they must go to root 'system' parameter)
	factsJSON, _ := json.Marshal(facts)
	messages = append(messages, anthropicMessage{
		Role: "user",
		Content: []anthropicContentPart{
			{Type: "text", Text: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON))},
		},
	})
	messages = append(messages, anthropicMessage{
		Role: "assistant",
		Content: []anthropicContentPart{
			{Type: "text", Text: "Acknowledged. I will cross-examine you and update these facts. Please provide your input."},
		},
	})

	// Add conversation history
	for _, m := range history {
		messages = append(messages, anthropicMessage{
			Role: m.Role,
			Content: []anthropicContentPart{
				{Type: "text", Text: m.Content},
			},
		})
	}

	// Add latest user input
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
		System:    systemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authApiKeyHeader, a.apiKey)
	req.Header.Set(anthropicVersionHeader, anthropicVersionValue)

	startTime := time.Now()
	respBytes, err := SendWithRetry(ctx, a.client, req, a.maxRetries)
	duration := time.Since(startTime)
	if err != nil {
		logger.LogAPI(config.ProviderAnthropic, a.model, duration, 0, 0, err)
		return nil, err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		logger.LogAPI(config.ProviderAnthropic, a.model, duration, 0, 0, err)
		return nil, fmt.Errorf(errParseAnthropicResponse, err)
	}

	if len(anthropicResp.Content) == 0 {
		errEmpty := fmt.Errorf(errEmptyContentAnthropic)
		logger.LogAPI(config.ProviderAnthropic, a.model, duration, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, errEmpty)
		return nil, errEmpty
	}

	var oracleResp OracleResponse
	contentStr := anthropicResp.Content[0].Text
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		errInvalidJSON := fmt.Errorf("Anthropic returned invalid Oracle JSON: %w (Raw content: %s)", err, contentStr)
		logger.LogAPI(config.ProviderAnthropic, a.model, duration, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, errInvalidJSON)
		return nil, errInvalidJSON
	}

	logger.LogAPI(config.ProviderAnthropic, a.model, duration, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens, nil)

	oracleResp.TokensPrompt = anthropicResp.Usage.InputTokens
	oracleResp.TokensCompletion = anthropicResp.Usage.OutputTokens
	oracleResp.NextQuestion = shared.SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (a *AnthropicGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := a.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	shared.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (a *AnthropicGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	messages := []anthropicMessage{
		{
			Role: "user",
			Content: []anthropicContentPart{
				{Type: "text", Text: promptTemplate},
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    GenerateSpecSystemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authApiKeyHeader, a.apiKey)
	req.Header.Set(anthropicVersionHeader, anthropicVersionValue)

	respBytes, err := SendWithRetry(ctx, a.client, req, a.maxRetries)
	if err != nil {
		return "", err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return "", fmt.Errorf(errParseAnthropicResponse, err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf(errEmptyContentAnthropic)
	}

	return anthropicResp.Content[0].Text, nil
}

func (a *AnthropicGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
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

	messages := []anthropicMessage{
		{
			Role: "user",
			Content: []anthropicContentPart{
				{Type: "text", Text: string(payloadBytes)},
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    systemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authApiKeyHeader, a.apiKey)
	req.Header.Set(anthropicVersionHeader, anthropicVersionValue)

	respBytes, err := SendWithRetry(ctx, a.client, req, a.maxRetries)
	if err != nil {
		return nil, err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return nil, fmt.Errorf(errParseAnthropicResponse, err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf(errEmptyContentAnthropic)
	}

	rawJSON := anthropicResp.Content[0].Text

	// Enforce parsing object wrappers
	if idx := strings.Index(rawJSON, "{"); idx != -1 {
		if endIdx := strings.LastIndex(rawJSON, "}"); endIdx != -1 && endIdx > idx {
			rawJSON = rawJSON[idx : endIdx+1]
		}
	}

	var envelope struct {
		Results []ComplianceResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		return nil, fmt.Errorf("Anthropic returned invalid compliance JSON: %w (Raw content: %s)", err, rawJSON)
	}

	return envelope.Results, nil
}

func (a *AnthropicGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
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

	messages := []anthropicMessage{
		{
			Role: "user",
			Content: []anthropicContentPart{
				{Type: "text", Text: prompt},
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    systemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authApiKeyHeader, a.apiKey)
	req.Header.Set(anthropicVersionHeader, anthropicVersionValue)

	respBytes, err := SendWithRetry(ctx, a.client, req, a.maxRetries)
	if err != nil {
		return "", err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return "", fmt.Errorf(errParseAnthropicResponse, err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf(errEmptyContentAnthropic)
	}

	return anthropicResp.Content[0].Text, nil
}

func (a *AnthropicGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	systemPrompt := ConsistencySystemPrompt

	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}

	payloadStruct := consistencyPayload{Files: files}
	payloadBytes, _ := json.Marshal(payloadStruct)

	messages := []anthropicMessage{
		{
			Role: "user",
			Content: []anthropicContentPart{
				{Type: "text", Text: string(payloadBytes)},
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		System:    systemPrompt,
		Messages:  messages,
		MaxTokens: 4000,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicChatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authApiKeyHeader, a.apiKey)
	req.Header.Set(anthropicVersionHeader, anthropicVersionValue)

	respBytes, err := SendWithRetry(ctx, a.client, req, a.maxRetries)
	if err != nil {
		return nil, err
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return nil, fmt.Errorf(errParseAnthropicResponse, err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf(errEmptyContentAnthropic)
	}

	contentStr := anthropicResp.Content[0].Text
	contentStr = shared.SanitizeJSON(contentStr)

	var report ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse consistency report JSON: %w (Raw content: %s)", err, contentStr)
	}

	return &report, nil
}
