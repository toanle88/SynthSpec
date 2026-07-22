package gateway

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/toanle/synthspec/domain"
)

// MockGateway implements the Gateway interface for local testing
type MockGateway struct {
	onTokenUsage func(prompt, completion int)
	budgetCheck  func() error
}

func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

func (m *MockGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string, currentScores ConfidenceScores, currentRationales DimensionRationales) (*OracleResponse, error) {
	turns := len(history)

	// Build updated facts by appending latest input to simulate LLM updating them
	updatedFacts := facts
	if latestInput != "" {
		if updatedFacts.Functional == "" {
			updatedFacts.Functional = "User wants: " + latestInput
		} else {
			updatedFacts.Functional += "\n- Added: " + latestInput
		}
		updatedFacts.Structural += "\n- System components determined from: " + latestInput
		updatedFacts.Security += "\n- Security considerations based on: " + latestInput
		updatedFacts.Compliance += "\n- Compliance requirements mapped from: " + latestInput
	}

	res := &OracleResponse{
		Facts:            updatedFacts,
		TokensPrompt:     120,
		TokensCompletion: 350,
	}

	switch {
	case turns < 2:
		res.ConfidenceScores = ConfidenceScores{Functional: 25, Structural: 15, Security: 10, Compliance: 5}
		res.NextQuestion = "What are the primary user roles and functional workflows of this application?"
		res.NextChoices = []string{
			"Standard Admin, Editor, Viewer roles",
			"E-commerce Buyer and Seller workflows",
			"SaaS Tenant Owner and Member workflows",
		}
		res.DimensionRationales = DimensionRationales{
			Functional: "Initial target vision is clear, but core user flows are undefined.",
			Structural: "High-level components are implied; database schema is unmapped.",
			Security:   "Basic auth is assumed but user session duration and details are missing.",
			Compliance: "Compliance scope (e.g. GDPR, local storage) is not established.",
		}
	case turns < 4:
		res.ConfidenceScores = ConfidenceScores{Functional: 60, Structural: 45, Security: 30, Compliance: 20}
		res.NextQuestion = "How do you plan to handle data storage, database transactions, and tenant isolation?"
		res.NextChoices = []string{
			"PostgreSQL with Schema-based tenant isolation",
			"MongoDB with Document-level tenant isolation",
			"SQLite for local-only single-tenant operation",
		}
		res.DimensionRationales = DimensionRationales{
			Functional: "Functional requirements are mostly mapped. User roles are clear.",
			Structural: "Database requirements identified, but table design is outstanding.",
			Security:   "Authentication mechanisms declared, but encryption at rest is unaddressed.",
			Compliance: "GDPR compliance constraints noted, but data deletion flows are unmapped.",
		}
	case turns < 6:
		res.ConfidenceScores = ConfidenceScores{Functional: 90, Structural: 80, Security: 70, Compliance: 60}
		res.NextQuestion = "What are the compliance and security threat boundaries, and are there any specific auditing requirements?"
		res.NextChoices = []string{
			"Strict SOC2/GDPR compliance with automated audit logs",
			"HIPAA compliance with encrypted storage and access control",
			"No formal external compliance boundaries required",
		}
		res.DimensionRationales = DimensionRationales{
			Functional: "Functional specs are complete.",
			Structural: "Relational database schema agreed upon; API routes are structured.",
			Security:   "JWT validation rules set. Threat model is missing input validation details.",
			Compliance: "Audit logs defined. Export and backup strategy needs alignment.",
		}
	default:
		const readyMsg = "100% complete. Ready for generation."
		res.ConfidenceScores = ConfidenceScores{Functional: 100, Structural: 100, Security: 100, Compliance: 100}
		res.NextQuestion = "" // Complete
		res.NextChoices = nil
		res.DimensionRationales = DimensionRationales{
			Functional: readyMsg,
			Structural: readyMsg,
			Security:   readyMsg,
			Compliance: readyMsg,
		}
	}

	res.NextQuestion = SanitizeNextQuestion(res.NextQuestion)
	return res, nil
}

func (m *MockGateway) QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, currentScores ConfidenceScores, currentRationales DimensionRationales, tokenChan chan<- string) (*OracleResponse, error) {
	res, err := m.QueryOracle(ctx, facts, history, latestInput, currentScores, currentRationales)
	if err != nil {
		close(tokenChan)
		return nil, err
	}
	domain.StreamOracleResponse(res, tokenChan)
	return res, nil
}

func (m *MockGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error) {
	return getMockGenerateSpecContent(fileName, facts)
}

// EvaluateCompliance returns mocked compliance scores matching the standards checklist
func (m *MockGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error) {
	var results []ComplianceResult

	for _, std := range standards {
		// Only check standards that target this file
		targetsFile := false
		for _, tf := range std.TargetFiles {
			if tf == fileName {
				targetsFile = true
				break
			}
		}
		if !targetsFile {
			continue
		}

		score := 0
		compliant := false
		feedback := ""

		switch std.ID {
		case "sql_parameterization", "soft_delete", "uuid_primary_keys", "timestamptz", "connection_pooling", "structured_logging", "prometheus_metrics", "cors", "theme_support", "directory_module_topography", "architectural_pattern_enforcement", "transport_protocol_standards", "contract_lifecycle_management", "domain_scenario_sequence_diagrams", "domain_bounded_contexts", "roadmap_phases_milestones", "roadmap_gantt_diagram":
			score = 100
			compliant = true
			feedback = fmt.Sprintf("Successfully implemented %s.", std.Name)
		case "clean_architecture":
			score = 70
			compliant = score >= std.MinScore
			feedback = "Clean architecture partial separation. Routing layers are slightly coupled."
		default:
			// Starts at 0%
			score = 0
			compliant = false
			feedback = fmt.Sprintf("Standard %s has not been implemented in the generated documentation.", std.Name)
		}

		// Simulating self-correction progress:
		// If the fileContent contains indicators of self-correction, bump the score to 100.
		if strings.Contains(fileContent, "Fix:") || strings.Contains(fileContent, "refined") || strings.Contains(fileContent, "compliant") {
			score = 100
			compliant = true
			feedback = fmt.Sprintf("Successfully refined standard: %s to 100%% compliance after self-correction.", std.Name)
		}

		results = append(results, ComplianceResult{
			StandardID: std.ID,
			Score:      score,
			Compliant:  compliant,
			Feedback:   feedback,
		})
	}

	return results, nil
}

func (m *MockGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error) {
	var ids []string
	for _, std := range failedStandards {
		ids = append(ids, std.ID)
	}
	fixMsg := fmt.Sprintf("refined Fix: compliant with %s", strings.Join(ids, ", "))

	if strings.TrimSpace(referenceDoc) != "" {
		return fmt.Sprintf("%s\n\n<!-- %s -->\n<!-- Reference source document preserved -->\n", fileContent, fixMsg), nil
	}
	return fmt.Sprintf("%s\n\n<!-- %s -->\n", fileContent, fixMsg), nil
}

func (m *MockGateway) VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error) {
	// By default, mock gateway reports that all files are logically consistent.
	// If a specific test trigger is present in the file content, mock an inconsistency.
	for fileName, content := range files {
		if strings.Contains(content, "TRIGGER_INCONSISTENCY") {
			return &ConsistencyReport{
				Consistent: false,
				Feedback: map[string]string{
					fileName: "Mock inconsistency detected: Please align definitions with standard schema.",
				},
			}, nil
		}
	}

	return &ConsistencyReport{
		Consistent: true,
		Feedback:   make(map[string]string),
	}, nil
}

// Summarize generates a concise summary of the conversation history for the mock gateway.
func (m *MockGateway) Summarize(ctx context.Context, history []Message) (string, error) {
	if len(history) == 0 {
		return "No conversation history to summarize.", nil
	}

	var summary strings.Builder
	summary.WriteString("Mock summary of conversation:\n")

	userTurns := 0
	for _, msg := range history {
		if msg.Role == "user" {
			userTurns++
			if userTurns <= 3 { // Only summarize first 3 user turns
				summary.WriteString(fmt.Sprintf("- User: %s\n", msg.Content))
			}
		}
	}

	if userTurns > 3 {
		summary.WriteString(fmt.Sprintf("- ... and %d more user messages\n", userTurns-3))
	}

	summary.WriteString(fmt.Sprintf("\nTotal turns: %d", len(history)))
	return summary.String(), nil
}

// ExtractStructuralEntities mock implementation
func (m *MockGateway) ExtractStructuralEntities(ctx context.Context, sourceDoc string) (string, error) {
	return `{"entities":[{"name":"MockEntity","attributes":["id","name"]}],"workflows":[{"name":"MockWorkflow","steps":["MockStep"]}],"integrations":[{"type":"MockIntegration","details":"MockDatabase"}]}`, nil
}

func (m *MockGateway) OptimizePrompt(ctx context.Context, files map[string]string) (string, error) {
	var sb strings.Builder
	sb.WriteString("# Mock Optimized Prompt\n\n")
	sb.WriteString("This is a mock optimized prompt generated from the following files:\n")
	for name := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", name))
	}
	sb.WriteString("\n## Directives\n")
	sb.WriteString("1. Implement all features according to the specifications.\n")
	return sb.String(), nil
}

// GenerateEmbeddings mock implementation using pseudoEmbed FNV-1a deterministic hashing
func (m *MockGateway) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i, txt := range texts {
		vec := make([]float32, 128)
		words := strings.Fields(strings.ToLower(txt))
		if len(words) == 0 {
			res[i] = vec
			continue
		}
		for d := 0; d < 128; d++ {
			var val float32
			for _, word := range words {
				h := uint32(2166136261)
				for j := 0; j < len(word); j++ {
					h = (h ^ uint32(word[j])) * 16777619
				}
				h = (h ^ uint32(d)) * 16777619
				val += float32(h) / float32(0xFFFFFFFF)
			}
			vec[d] = val / float32(len(words))
		}

		// Normalize
		var norm float64
		for _, v := range vec {
			norm += float64(v * v)
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for idx := range vec {
				vec[idx] = float32(float64(vec[idx]) / norm)
			}
		}
		res[i] = vec
	}
	return res, nil
}

func (m *MockGateway) RegisterTokenCounter(fn func(prompt, completion int)) {
	m.onTokenUsage = fn
}

func (m *MockGateway) RegisterBudgetCheck(fn func() error) {
	// MockGateway typically doesn't enforce budget limits unless specifically tested
}
