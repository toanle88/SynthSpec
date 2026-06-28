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
	"github.com/toanle/synthspec/logger"
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
	oracleResp.NextQuestion = SanitizeNextQuestion(oracleResp.NextQuestion)

	return &oracleResp, nil
}

func (a *AnthropicGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := a.QueryOracle(ctx, facts, history, latestInput)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	StreamOracleResponse(res, tokenChan)
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
		System:    "You are a senior solutions architect. Write detailed, enterprise-grade specification files based on the facts provided. Return the exact file content and nothing else. No preamble, no postamble, no markdown codeblocks unless specified.",
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

func (a *AnthropicGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []config.Standard) ([]ComplianceResult, error) {
	applicableStandards := FilterApplicableStandards(standards, fileName)

	if len(applicableStandards) == 0 {
		return nil, nil
	}

	systemPrompt := `You are an expert software engineering auditor. Your job is to evaluate if a generated specification file complies with specific architectural and software development standards.
For each standard provided, evaluate the file content and return a JSON object with a root key "results" containing an array of evaluation objects.
Each evaluation object must contain:
1. "standard_id": the ID of the standard being evaluated.
2. "score": an integer from 0 to 100 indicating compliance (0 for completely absent/fails, 100 for fully compliant).
3. "compliant": a boolean indicating if it meets the minimum threshold or is acceptable.
4. "feedback": a concise explanation of the score and specific details of what is missing or incorrect.

Your response MUST be a JSON object matching this structure:
{
  "results": [
    {
      "standard_id": "clean_architecture",
      "score": 75,
      "compliant": true,
      "feedback": "Decoupling is partially complete..."
    }
  ]
}
Output only the raw JSON string. Do NOT output any markdown formatting backticks.`

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

func (a *AnthropicGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []config.Standard, referenceDoc string) (string, error) {
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
%s

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
	systemPrompt := `You are an expert software engineering auditor. Your job is to verify that all generated specification files are logically consistent with one another.
Compare functional requirements, API endpoints, data models, compliance specifications, and system architectures.
Analyze the provided documents and output:
1. "consistent": a boolean indicating whether all files are fully consistent with zero contradictions.
2. "feedback": a map of filename key to string value detailing the discrepancy/correction instructions. Only include files in this map that have errors/inconsistencies. If consistent is true, this map must be empty.

Your response MUST be a JSON object, like this:
{
  "consistent": false,
  "feedback": {
    "04_api_architecture_integration.md": "Rename the /users endpoint to /accounts to match the system architecture document."
  }
}
Do NOT return markdown code block backticks. Output only the raw JSON string.`

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
	contentStr = sanitizeJSON(contentStr)

	var report ConsistencyReport
	if err := json.Unmarshal([]byte(contentStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse consistency report JSON: %w (Raw content: %s)", err, contentStr)
	}

	return &report, nil
}

