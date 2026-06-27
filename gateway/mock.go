package gateway

import (
	"context"
	"encoding/json"
	"fmt"
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
