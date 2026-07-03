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
	geminiChatURLTemplate   = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
	errParseGeminiResponse  = "failed to parse Gemini response: %w"
	errEmptyCandidateGemini = "empty response candidate returned from Gemini"
)

type GeminiGateway struct {
	apiKey     string
	model      string
	client     *http.Client
	maxRetries int
}

func NewGeminiGateway(apiKey, model string) *GeminiGateway {
	if model == "" {
		model = "gemini-2.5-pro"
	}
	timeout := 5 * time.Minute
	maxRetries := 3

	if s, err := config.LoadSettings(); err == nil && s != nil {
		timeout = time.Duration(s.TimeoutSeconds) * time.Second
		maxRetries = s.MaxRetries
	}

	return &GeminiGateway{
		apiKey:     apiKey,
		model:      model,
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
}

// Gemini API structures
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" or "model"
	Parts []geminiPart `json:"parts"`
}

type geminiInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type geminiRequest struct {
	SystemInstruction *geminiInstruction `json:"systemInstruction,omitempty"`
	Contents          []geminiContent    `json:"contents"`
	GenerationConfig  *geminiConfig      `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (g *GeminiGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
	systemPrompt := OracleSystemPrompt

	contents := []geminiContent{}

	// Add facts context as user content to begin the prompt context
	factsJSON, _ := json.Marshal(facts)
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON))}},
	})
	contents = append(contents, geminiContent{
		Role:  "model",
		Parts: []geminiPart{{Text: "Acknowledged. Ready to receive user answers and cross-examine."}},
	})

	// Add conversation history
	for _, m := range history {
		role := m.Role
		if role == "assistant" {
			role = "model"
		} else {
			role = "user"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	// Add latest user input
	if latestInput != "" {
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: latestInput}},
		})
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: contents,
		GenerationConfig: &geminiConfig{
			ResponseMimeType: "application/json",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	startTime := time.Now()
	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	duration := time.Since(startTime)
	if err != nil {
		logger.LogAPI(config.ProviderGemini, g.model, duration, 0, 0, err)
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		logger.LogAPI(config.ProviderGemini, g.model, duration, 0, 0, err)
		return nil, fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		errEmpty := fmt.Errorf(errEmptyCandidateGemini)
		logger.LogAPI(config.ProviderGemini, g.model, duration, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, errEmpty)
		return nil, errEmpty
	}

	var oracleResp OracleResponse
	contentStr := geminiResp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		errInvalidJSON := fmt.Errorf("Gemini returned invalid Oracle JSON: %w (Raw content: %s)", err, contentStr)
		logger.LogAPI(config.ProviderGemini, g.model, duration, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, errInvalidJSON)
		return nil, errInvalidJSON
	}

	logger.LogAPI(config.ProviderGemini, g.model, duration, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, nil)

	oracleResp.TokensPrompt = geminiResp.UsageMetadata.PromptTokenCount
	oracleResp.TokensCompletion = geminiResp.UsageMetadata.CandidatesTokenCount
	oracleResp.NextQuestion = shared.SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (g *GeminiGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := g.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	shared.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (g *GeminiGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: promptTemplate}},
		},
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: GenerateSpecSystemPrompt}},
		},
		Contents: contents,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf(errEmptyCandidateGemini)
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
	applicableStandards := config.FilterApplicableStandards(standards, fileName)

	if len(applicableStandards) == 0 {
		return nil, nil
	}

	systemPrompt := ComplianceSystemPromptArray

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

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: string(payloadBytes)}},
		},
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: contents,
		GenerationConfig: &geminiConfig{
			ResponseMimeType: applicationJSON,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	if err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf(errEmptyCandidateGemini)
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	// Enforce trimming block wrappers if returned in text
	if idx := strings.Index(rawJSON, "["); idx != -1 {
		if endIdx := strings.LastIndex(rawJSON, "]"); endIdx != -1 && endIdx > idx {
			rawJSON = rawJSON[idx : endIdx+1]
		}
	}

	var results []ComplianceResult
	if err := json.Unmarshal([]byte(rawJSON), &results); err != nil {
		return nil, fmt.Errorf("Gemini returned invalid compliance JSON: %w (Raw content: %s)", err, rawJSON)
	}

	return results, nil
}

func (g *GeminiGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
	systemPrompt := "You are a senior solutions architect. Your job is to modify an existing specification file to fix quality standards violations. Return only the updated file contents and nothing else. No preamble, no postamble, no markdown codeblocks unless specified."

	// Format standard criteria to make instructions clear
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

Reference source RefineSystemPrompt
%s

Original File Content:
%s

CRITICAL: When rewriting this file to fix the audit failures, do not abbreviate, truncate, or omit any existing sections that are already passing. You must maintain or improve the detail level of the entire document.

Return ONLY the updated file contents. Do NOT wrap it in markdown code blocks like `+"```"+` or include any conversational filler.`,
		fileName, feedback, criteriaText, strings.TrimSpace(referenceDoc), fileContent)

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: prompt}},
		},
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: contents,
	}

	if fileName == "05_engineering_backlog.json" {
		reqBody.GenerationConfig = &geminiConfig{
			ResponseMimeType: applicationJSON,
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf(errEmptyCandidateGemini)
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	systemPrompt := ConsistencySystemPrompt

	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}

	payloadStruct := consistencyPayload{Files: files}
	payloadBytes, _ := json.Marshal(payloadStruct)

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: string(payloadBytes)}},
		},
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: contents,
		GenerationConfig: &geminiConfig{
			ResponseMimeType: applicationJSON,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	if err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf(errEmptyCandidateGemini)
	}

	contentStr := geminiResp.Candidates[0].Content.Parts[0].Text
	contentStr = shared.SanitizeJSON(contentStr)

	var report ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse consistency report JSON: %w (Raw content: %s)", err, contentStr)
	}

	return &report, nil
}

// Summarize generates a concise summary of the conversation history using the Gemini model.
func (g *GeminiGateway) Summarize(ctx context.Context, history []Message) (string, error) {
	systemPrompt := "You are a technical summarizer. Compress the conversation history into a single clear paragraph summarizing the key architectural decisions, user preferences, and engineering requirements established. Focus on consensus outcomes, not the back-and-forth dialogue."

	contents := []geminiContent{}

	// Add conversation history
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	// Add summarization instruction
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: "Summarize the above conversation into a single paragraph capturing the key decisions and requirements."}},
	})

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: contents,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)

	respBytes, err := SendWithRetry(ctx, g.client, req, g.maxRetries)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf(errEmptyCandidateGemini)
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
