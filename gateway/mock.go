package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toanle/synthspec/config"
)

// MockGateway implements the Gateway interface for local testing
type MockGateway struct{}

func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

func (m *MockGateway) QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error) {
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
		Facts: updatedFacts,
		TokensPrompt: 120,
		TokensCompletion: 350,
	}

	switch {
	case turns < 2:
		res.ConfidenceScores = ConfidenceScores{Functional: 25, Structural: 15, Security: 10, Compliance: 5}
		res.NextQuestion = "What are the primary user roles and functional workflows of this application?"
		res.DimensionRationales = DimensionRationales{
			Functional: "Initial target vision is clear, but core user flows are undefined.",
			Structural: "High-level components are implied; database schema is unmapped.",
			Security:   "Basic auth is assumed but user session duration and details are missing.",
			Compliance: "Compliance scope (e.g. GDPR, local storage) is not established.",
		}
	case turns < 4:
		res.ConfidenceScores = ConfidenceScores{Functional: 60, Structural: 45, Security: 30, Compliance: 20}
		res.NextQuestion = "How do you plan to handle data storage, database transactions, and tenant isolation?"
		res.DimensionRationales = DimensionRationales{
			Functional: "Functional requirements are mostly mapped. User roles are clear.",
			Structural: "Database requirements identified, but table design is outstanding.",
			Security:   "Authentication mechanisms declared, but encryption at rest is unaddressed.",
			Compliance: "GDPR compliance constraints noted, but data deletion flows are unmapped.",
		}
	case turns < 6:
		res.ConfidenceScores = ConfidenceScores{Functional: 90, Structural: 80, Security: 70, Compliance: 60}
		res.NextQuestion = "What are the compliance and security threat boundaries, and are there any specific auditing requirements?"
		res.DimensionRationales = DimensionRationales{
			Functional: "Functional specs are complete.",
			Structural: "Relational database schema agreed upon; API routes are structured.",
			Security:   "JWT validation rules set. Threat model is missing input validation details.",
			Compliance: "Audit logs defined. Export and backup strategy needs alignment.",
		}
	default:
		res.ConfidenceScores = ConfidenceScores{Functional: 100, Structural: 100, Security: 100, Compliance: 100}
		res.NextQuestion = "" // Complete
		res.DimensionRationales = DimensionRationales{
			Functional: "100% complete. Ready for generation.",
			Structural: "100% complete. Ready for generation.",
			Security:   "100% complete. Ready for generation.",
			Compliance: "100% complete. Ready for generation.",
		}
	}

	res.NextQuestion = SanitizeNextQuestion(res.NextQuestion)
	return res, nil
}

func (m *MockGateway) GenerateSpecFile(ctx context.Context, facts Facts, fileName string) (string, error) {
	switch fileName {
	case "01_prd_functional.md":
		return fmt.Sprintf(`# Functional Requirements Document (PRD)

* **Status**: 🟢 Approved
* **Author**: SynthSpec (Mock Engine)

## 🎯 Product Vision
This specification outlines the functional features compiled during the interrogation loop.

## 📋 Compiled User Input
%s

## ⚙️ Core Features
1. **User Authentication & Management**: Role-based access control.
2. **Interactive Workspace**: Local state and session caching.
3. **Audit Trails**: Capture details of user operations.
`, facts.Functional), nil

	case "02_system_architecture.md":
		return fmt.Sprintf(`# System Architecture Specification

* **Status**: 🟢 Approved

## 🏗️ Architectural Topology
The application is structured as a decoupled, layered system.

` + "```mermaid" + `
graph TD
    Client[CLI Terminal Client] -->|API Calls| API[API Routing Layer]
    API -->|Storage Interface| DB[(Relational DB)]
    API -->|Logs| Audit[Audit Trail Service]
` + "```" + `

## 💾 Compiled Structural Facts
%s
`, facts.Structural), nil

	case "03_security_threat_model.md":
		return fmt.Sprintf(`# Security & STRIDE Threat Model

* **Status**: 🟢 Approved

## 🛡️ Threats and Mitigations

| Category | Vulnerability | Mitigation Strategy |
|----------|---------------|---------------------|
| **Spoofing** | Unauthorized API access | Cryptographic JWT claims and signature validation. |
| **Information Disclosure** | Leakage of tenant data | Query-level row separation and parameter validation. |

## 🔒 Compiled Security Facts
%s
`, facts.Security), nil

	case "04_openapi_contract.yaml":
		return `openapi: 3.0.3
info:
  title: SynthSpec Mock API
  version: 1.0.0
  description: Automatically generated OpenAPI contract.
paths:
  /api/v1/auth:
    post:
      summary: Authenticate User
      responses:
        '200':
          description: Successful login
  /api/v1/projects:
    get:
      summary: List Projects
      responses:
        '200':
          description: List of projects
`, nil

	case "05_engineering_backlog.json":
		backlog := map[string]interface{}{
			"epics": []map[string]interface{}{
				{
					"id":          "EPIC-001",
					"title":       "Core Foundation",
					"description": "Establish basic project structure and authentication middleware.",
					"tasks": []map[string]interface{}{
						{
							"id":       "TSK-101",
							"summary":  "Setup database migrations",
							"details":  "Establish schema files and table structures.",
							"acceptance_criteria": []string{
								"All migrations run successfully in local dev environment.",
								"Rollback scripts verified.",
							},
						},
						{
							"id":       "TSK-102",
							"summary":  "Implement JWT authentication middleware",
							"details":  "Verify authorization headers and extract tenant context.",
							"acceptance_criteria": []string{
								"Reject requests with missing or expired tokens with 401.",
								"Inject verified tenant ID into request context.",
							},
						},
					},
				},
			},
		}
		data, err := json.MarshalIndent(backlog, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown file: %s", fileName)
	}
}

// EvaluateCompliance returns mocked compliance scores matching the standards checklist
func (m *MockGateway) EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []config.Standard) ([]ComplianceResult, error) {
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
		case "sql_parameterization", "soft_delete", "uuid_primary_keys", "timestamptz", "connection_pooling", "structured_logging", "prometheus_metrics", "cors", "theme_support":
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

func (m *MockGateway) RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []config.Standard) (string, error) {
	var ids []string
	for _, std := range failedStandards {
		ids = append(ids, std.ID)
	}
	fixMsg := fmt.Sprintf("refined Fix: compliant with %s", strings.Join(ids, ", "))

	if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
		return fmt.Sprintf("%s\n# %s\n", fileContent, fixMsg), nil
	}

	if strings.HasSuffix(fileName, ".json") {
		content := strings.TrimSpace(fileContent)
		if strings.HasPrefix(content, "```") {
			if idx := strings.Index(content, "\n"); idx != -1 {
				content = content[idx+1:]
			}
			if strings.HasSuffix(content, "```") {
				content = content[:len(content)-3]
			}
			content = strings.TrimSpace(content)
		}

		var jsonObj map[string]interface{}
		if err := json.Unmarshal([]byte(content), &jsonObj); err == nil {
			jsonObj["compliance_refinement"] = fixMsg
			if bytes, marshalErr := json.MarshalIndent(jsonObj, "", "  "); marshalErr == nil {
				return string(bytes), nil
			}
		}
		return fileContent, nil
	}

	return fmt.Sprintf("%s\n\n<!-- %s -->\n", fileContent, fixMsg), nil
}

