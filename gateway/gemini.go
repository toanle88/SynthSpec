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
)

type GeminiGateway struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiGateway(apiKey, model string) *GeminiGateway {
	if model == "" {
		model = "gemini-2.5-pro"
	}
	return &GeminiGateway{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
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
	systemPrompt := `You are SynthSpec, an expert AI Solution Architect. Your goal is to help the user build an enterprise-grade engineering specification.
You operate in a strict single-question interrogation loop, cross-examining the user.

Your response MUST be a single valid JSON object matching the following structure:
{
  "facts": {
    "functional": "Detailed summary of all functional features, workflows, and user roles agreed on so far.",
    "structural": "Detailed summary of structural/architectural preferences (e.g. database, language, communication protocols).",
    "security": "Detailed summary of security constraints (e.g. authentication, JWT, encryption, threat limits).",
    "compliance": "Detailed summary of compliance rules (e.g. tenancy model, GDPR, data retention)."
  },
  "confidence_scores": {
    "functional": 0 to 100 integer,
    "structural": 0 to 100 integer,
    "security": 0 to 100 integer,
    "compliance": 0 to 100 integer
  },
  "next_question": "Exactly ONE question targeting missing details. Leave empty if ALL scores are 100.",
  "next_choices": ["Option 1", "Option 2", "Option 3"],
  "dimension_rationales": {
    "functional": "Why did you assign this functional score?",
    "structural": "Why did you assign this structural score?",
    "security": "Why did you assign this security score?",
    "compliance": "Why did you assign this compliance score?"
  }
}

Guidelines for next_choices:
- Under "next_choices", provide a JSON array of 3-5 concise, specific choice options that directly answer "next_question".
- Put the most recommended or industry-standard option as the first item in the array.
- Leave this array empty if "next_question" is empty.

Guidelines for evaluation:
- Be strict. Do not give 100% confidence on any dimension until the specific requirements are clear and complete.
- Functional is complete when user roles, core workflows, and at least 3-4 key features are clarified.
- Structural is complete when the database choice, API schema, backend/frontend stacks are specified.
- Security is complete when authentication, authorization (RBAC), and encryption methods are defined.
- Compliance is complete when tenancy model (multi-tenant vs single-tenant), GDPR/data-handling, and backup strategies are set.
- Under NO circumstances ask more than ONE question at a time. Do not use bullets or lists for questions; ask a single clear question.
- Do NOT output any markdown backticks wrapper (like ` + "```json" + `). Output ONLY the raw JSON string.`

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

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	respBytes, err := SendWithRetry(ctx, g.client, req, 3)
	if err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response candidate returned from Gemini")
	}

	var oracleResp OracleResponse
	contentStr := geminiResp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(contentStr), &oracleResp); err != nil {
		return nil, fmt.Errorf("Gemini returned invalid Oracle JSON: %w (Raw content: %s)", err, contentStr)
	}

	oracleResp.TokensPrompt = geminiResp.UsageMetadata.PromptTokenCount
	oracleResp.TokensCompletion = geminiResp.UsageMetadata.CandidatesTokenCount
	oracleResp.NextQuestion = SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (g *GeminiGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string) (string, error) {
	var prompt string
	switch fileName {
	case "01_prd_functional.md":
		prompt = "Write a comprehensive Product Requirements Document (PRD) markdown file. Include product vision, user stories, features list, and a detailed functional requirements matrix with ID, Feature Name, Description, and Acceptance Criteria. Use these facts:\n"
	case "02_system_architecture.md":
		prompt = "Write a high-level System Architecture specification markdown file. Detail the component layout, backend layer division, API routing logic, database schema design (include raw SQL tables), and a Mermaid.js diagram showing workflow sequence/architecture. Use these facts:\n"
	case "03_security_threat_model.md":
		prompt = "Write a detailed Security & Threat Model markdown file. Perform a STRIDE threat modeling analysis. Map identified threats (at least 5) to mitigations in a clean markdown table. Detail input validation, timeout configurations, and cryptographic standards. Use these facts:\n"
	case "04_openapi_contract.yaml":
		prompt = "Write a complete, valid OpenAPI v3.0 REST API specification contract in YAML format. It must outline authentications, request parameters, response models, error states, and endpoints for core workflows. Do NOT include markdown backticks. Output ONLY the raw YAML. Use these facts:\n"
	case "05_engineering_backlog.json":
		prompt = `Generate a valid JSON document matching the Engineering Backlog schema.
The schema requires an object with a root key "epics" containing an array of epics. Each epic has id, title, description, and "tasks". Each task has id, summary, details, and "acceptance_criteria" (array of strings).
Do NOT include markdown backticks. Output ONLY the raw JSON. Use these facts:
`
	default:
		return "", fmt.Errorf("unknown file: %s", fileName)
	}

	factsJSON, _ := json.MarshalIndent(facts, "", "  ")
	fullPrompt := prompt + string(factsJSON)

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: fullPrompt}},
		},
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiInstruction{
			Parts: []geminiPart{{Text: "You are a senior solutions architect. Write detailed, enterprise-grade specification files based on the facts provided. Return the exact file content and nothing else. No preamble, no postamble, no markdown codeblocks unless specified."}},
		},
		Contents: contents,
	}

	// For JSON file, enforce json output
	if fileName == "05_engineering_backlog.json" {
		reqBody.GenerationConfig = &geminiConfig{
			ResponseMimeType: "application/json",
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	respBytes, err := SendWithRetry(ctx, g.client, req, 3)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response candidate returned from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []config.Standard) ([]ComplianceResult, error) {
	var applicableStandards []config.Standard
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

	systemPrompt := `You are an expert software engineering auditor. Your job is to evaluate if a generated specification file complies with specific architectural and software development standards.
For each standard provided, evaluate the file content and return:
1. "standard_id": the ID of the standard being evaluated.
2. "score": an integer from 0 to 100 indicating compliance (0 for completely absent/fails, 100 for fully compliant).
3. "compliant": a boolean indicating if it meets the minimum threshold or is acceptable.
4. "feedback": a concise explanation of the score and specific details of what is missing or incorrect.

Your response MUST be a JSON array of objects representing these evaluation results, like this:
[
  {
    "standard_id": "clean_architecture",
    "score": 75,
    "compliant": true,
    "feedback": "Decoupling is partially complete..."
  }
]
Do NOT return markdown code block backticks. Output only the raw JSON array string.`

	type auditPayload struct {
		FileName    string            `json:"file_name"`
		FileContent string            `json:"file_content"`
		Standards   []config.Standard `json:"standards"`
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
			ResponseMimeType: "application/json",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	respBytes, err := SendWithRetry(ctx, g.client, req, 3)
	if err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response candidate returned from Gemini")
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

func (g *GeminiGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []config.Standard) (string, error) {
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

Original File Content:
%s

Return ONLY the updated file contents. Do NOT wrap it in markdown code blocks like `+"```"+` or include any conversational filler.`,
		fileName, feedback, criteriaText, fileContent)

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

	// For JSON file, enforce json output
	if fileName == "05_engineering_backlog.json" {
		reqBody.GenerationConfig = &geminiConfig{
			ResponseMimeType: "application/json",
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	respBytes, err := SendWithRetry(ctx, g.client, req, 3)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response candidate returned from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

