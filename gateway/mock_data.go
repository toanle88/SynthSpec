package gateway

import "fmt"

func getMockGenerateSpecContent(fileName string, facts Facts) (string, error) {
	switch fileName {
	case "01_domain_model_use_cases.md":
		return `# Domain Model & Use Cases

* **Status**: 🟢 Approved

## 🗺️ Bounded Context Map
Account Management Context, Spec Synthesis Context.

## 🧱 Core Domain Entities & Value Objects
Entities: Project, Requirement, Standard.
Value Objects: Score, Rationale.

## 🚀 Scenario Walkthroughs
Primary use case scenario details.
`, nil

	case "02_prd_functional.md":
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

	case "03_system_architecture.md":
		return fmt.Sprintf(`# System Architecture Specification

* **Status**: 🟢 Approved

## 🏗️ Backend Topography
Three-tier architecture with load balancer.

## 📦 System Database Model
%s

## 🔀 API Routing Logic
REST endpoints map to specific service controllers.
`, facts.Structural), nil

	case "04_api_architecture_integration.md":
		return `# API Architecture & Integration Guide

* **Status**: 🟢 Approved

## 🌐 Transport & Protocol Standards
RESTful routing over HTTPS. JSON payload format.

## 🔄 Contract Lifecycle Management
Semantic versioning inside URL prefix.

## 📦 Global Payload Serialization
ISO 8601 timestamps and camelCase naming conventions.

## 🛠️ Cross-Cutting Concerns
Validation and rate limiting enabled.
`, nil

	case "05_coding_standards_guidelines.md":
		return `# Coding Standards & Guidelines

* **Status**: 🟢 Approved

## 📂 Directory & Module Topography
Visual directory layout and layer division.

## 🏗️ Architectural Pattern Enforcement
Strict dependency injection and repository patterns.

## 🧪 Testing Strategy & Coverage Gates
Integration testing with mock interfaces. 80% code coverage gate.

## 🧹 Linting & Static Analysis Rules
Configured strict rules.
`, nil

	case "06_security_threat_model.md":
		return fmt.Sprintf(`# Security & Threat Model

* **Status**: 🟢 Approved

## 🛡️ STRIDE Threat Assessment
| Category | Vulnerability | Mitigation Strategy |
|----------|---------------|---------------------|
| **Spoofing** | Unauthorized API access | Cryptographic JWT claims and signature validation. |
| **Information Disclosure** | Leakage of tenant data | Query-level row separation and parameter validation. |

## 🔒 Compiled Security Facts
%s
`, facts.Security), nil

	case "07_engineering_roadmap.md":
		return `# Engineering Roadmap

* **Status**: 🟢 Approved

## 🗺️ Phase-by-Phase Execution
- Phase 1 MVP: Basic CLI and Core Logic.
- Phase 2 Scale: Additional Gateway APIs.
- Phase 3 Future: Enterprise Web UI.

## 📅 Milestones
1. MVP release in Q3.
2. Web UI beta in Q4.

## 📊 Gantt Timeline
` + "```mermaid" + `
gantt
    title Project Timeline
    section Phase 1
    MVP :active, 2026-07-01, 30d
` + "```" + `
`, nil

	default:
		return "", fmt.Errorf("unknown file: %s", fileName)
	}
}
