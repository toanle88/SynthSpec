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
	geminiChatURLTemplate   = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
	errParseGeminiResponse  = "failed to parse Gemini response: %w"
	errEmptyCandidateGemini = "empty response candidate returned from Gemini"
)

type GeminiAdapter struct {
	apiKey string
	model  string
}

func NewGeminiAdapter(apiKey, model string) *GeminiAdapter {
	if model == "" {
		model = "gemini-2.5-pro"
	}
	return &GeminiAdapter{
		apiKey: apiKey,
		model:  model,
	}
}

func (g *GeminiAdapter) ProviderName() string {
	return config.ProviderGemini
}

func (g *GeminiAdapter) ModelName() string {
	return g.model
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



func (g *GeminiAdapter) BuildOracleRequest(facts domain.Facts, history []domain.Message, latestInput string) (*http.Request, error) {
	contents := []geminiContent{}

	factsJSON, _ := json.Marshal(facts)
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: fmt.Sprintf("Current compiled facts:\n%s", string(factsJSON))}},
	})
	contents = append(contents, geminiContent{
		Role:  "model",
		Parts: []geminiPart{{Text: "Acknowledged. Ready to receive user answers and cross-examine."}},
	})

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

	if latestInput != "" {
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: latestInput}},
		})
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: OracleSystemPrompt}},
		},
		Contents: contents,
		GenerationConfig: &geminiConfig{
			ResponseMimeType: applicationJSON,
		},
	}

	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) ParseOracleResponse(body []byte) (*domain.OracleResponse, int, int, error) {
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseGeminiResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, fmt.Errorf(errEmptyCandidateGemini)
	}

	contentStr := geminiResp.Candidates[0].Content.Parts[0].Text
	var oracleResp domain.OracleResponse
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		return nil, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, fmt.Errorf("invalid Oracle JSON: %w", err)
	}
	return &oracleResp, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, nil
}

func (g *GeminiAdapter) BuildGenerateSpecRequest(facts domain.Facts, fileName string, promptTemplate string) (*http.Request, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: GenerateSpecSystemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: promptTemplate}}},
		},
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) BuildExtractStructuralEntitiesRequest(sourceDoc string) (*http.Request, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: EntityExtractionSystemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: sourceDoc}}},
		},
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}


func (g *GeminiAdapter) ParseGenerateSpecResponse(body []byte) (string, int, int, error) {
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", 0, 0, fmt.Errorf(errParseGeminiResponse, err)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", 0, 0, fmt.Errorf(errEmptyCandidateGemini)
	}
	return geminiResp.Candidates[0].Content.Parts[0].Text, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, nil
}

func (g *GeminiAdapter) BuildEvaluateComplianceRequest(fileName string, fileContent string, standards []domain.Standard) (*http.Request, error) {
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

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: ComplianceSystemPromptArray}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: string(payloadBytes)}}},
		},
		GenerationConfig: &geminiConfig{
			ResponseMimeType: applicationJSON,
		},
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) ParseEvaluateComplianceResponse(body []byte) ([]domain.ComplianceResult, int, int, error) {
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseGeminiResponse, err)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyCandidateGemini)
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text
	if idx := strings.Index(rawJSON, "["); idx != -1 {
		if endIdx := strings.LastIndex(rawJSON, "]"); endIdx != -1 && endIdx > idx {
			rawJSON = rawJSON[idx : endIdx+1]
		}
	}

	var results []domain.ComplianceResult
	if err := json.Unmarshal([]byte(rawJSON), &results); err != nil {
		return nil, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, fmt.Errorf("invalid compliance JSON: %w", err)
	}
	return results, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, nil
}

func (g *GeminiAdapter) BuildRefineSpecRequest(fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (*http.Request, error) {
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

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: RefineSystemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
	}
	if fileName == "05_engineering_backlog.json" {
		reqBody.GenerationConfig = &geminiConfig{
			ResponseMimeType: applicationJSON,
		}
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) ParseRefineSpecResponse(body []byte) (string, int, int, error) {
	return g.ParseGenerateSpecResponse(body)
}

func (g *GeminiAdapter) BuildVerifyConsistencyRequest(files map[string]string) (*http.Request, error) {
	type consistencyPayload struct {
		Files map[string]string `json:"files"`
	}
	payloadBytes, _ := json.Marshal(consistencyPayload{Files: files})

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: ConsistencySystemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: string(payloadBytes)}}},
		},
		GenerationConfig: &geminiConfig{
			ResponseMimeType: applicationJSON,
		},
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) ParseVerifyConsistencyResponse(body []byte) (*domain.ConsistencyReport, int, int, error) {
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, 0, 0, fmt.Errorf(errParseGeminiResponse, err)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, 0, 0, fmt.Errorf(errEmptyCandidateGemini)
	}

	contentStr := SanitizeJSON(geminiResp.Candidates[0].Content.Parts[0].Text)
	var report domain.ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, fmt.Errorf("failed to parse consistency report JSON: %w", err)
	}
	return &report, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount, nil
}

func (g *GeminiAdapter) BuildSummarizeRequest(history []domain.Message) (*http.Request, error) {
	contents := []geminiContent{}
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
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: "Summarize the above conversation into a single paragraph capturing the key decisions and requirements."}},
	})

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: "You are a technical summarizer. Compress the conversation history into a single clear paragraph summarizing the key architectural decisions, user preferences, and engineering requirements established. Focus on consensus outcomes, not the back-and-forth dialogue."}},
		},
		Contents: contents,
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}

func (g *GeminiAdapter) ParseSummarizeResponse(body []byte) (string, int, int, error) {
	return g.ParseGenerateSpecResponse(body)
}

func (g *GeminiAdapter) BuildOptimizePromptRequest(files map[string]string) (*http.Request, error) {
	var sb strings.Builder
	for name, content := range files {
		sb.WriteString(fmt.Sprintf("=== FILE: %s ===\n%s\n\n", name, content))
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: OptimizePromptSystemPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: sb.String()}}},
		},
	}
	return buildJSONRequest(fmt.Sprintf(geminiChatURLTemplate, g.model, g.apiKey), reqBody, nil)
}
