# SynthSpec Documentation Overview

Welcome to the SynthSpec project documentation. This workspace contains technical specifications, architecture diagrams, guidelines, and standards for developers and maintainers of the SynthSpec CLI tool.

## Document Directory Map

### 1. [Architecture Design](architecture/system.md)
Detailed breakdowns of the CLI's internal event-driven architecture, models, and component boundaries.
- **[System & Component Design](architecture/system.md)**: Entrypoint, State Controller, The Architect persona, and overall architecture graph.
- **[TUI Dashboard Engine](architecture/tui.md)**: Asynchronous UI rendering, layouts, input mapping, and The Architect persona integration.
- **[LLM Gateway](architecture/gateway.md)**: Multi-model routing, token optimization, and exponential backoff retry algorithms.
- **Architecture Decision Records (ADRs)**:
  - **[001 Layered Architecture](architecture/decisions/001-layered-architecture.md)**: Layered structure definitions and rules.
  - **[002 Gateway Interface](architecture/decisions/002-gateway-interface.md)**: Standardized LLM provider integrations.
  - **[003 Source-First Generation](architecture/decisions/003-source-first-generation.md)**: Generation flow from domain specification source files.
  - **[004 Session Persistence Decoupling](architecture/decisions/004-session-persistence-decoupling.md)**: Storing state independent of interactive loops.
  - **[005 TUI Bubbletea](architecture/decisions/005-tui-bubbletea.md)**: Choosing Bubble Tea framework for terminal user interfaces.

### 2. [Functional Specifications](spec/README.md)
Detailed product requirements and workflows mapped out.
- **[Product Requirements Document (PRD)](spec/product.md)**: Product requirements, functional requirements matrix, STRIDE threat modeling analysis, and output schemas.
- **[The Oracle Interrogation Loop](spec/interrogation_loop.md)**: The interactive single-question loop, confidence scores, and verification gates.
- **[Asset Generation](spec/asset_generation.md)**: Source-doc-first generation, parallel downstream fan-out, output workspaces, and schemas.

### 3. Development & Maintenance
Guides and policies for developers contributing to SynthSpec.
- **[Quickstart Guide](development/quickstart.md)**: Get up and running with SynthSpec development in 5 minutes.
- **[Go Coding Standards](development/coding-go.md)**: Directory structures, conventions, and style constraints.
- **[General Coding Standards](development/coding.md)**: Local-first design, commit messages, and version control policies.
- **[Error Handling Policies](development/errors.md)**: API transport wrapping, TUI crash recovery, and state file serialization safeguards.
- **[Testing Strategies](development/testing-go.md)**: Go unit tests, mocking, and CLI integration tests.
- **[General Testing Standards](development/testing.md)**: Mock providers, deterministic outputs, and CI/CD coverage thresholds.
- **[Contributing Guide](development/contributing.md)**: Workflow guidelines, code of conduct, and pull request processes.

### 4. Infrastructure & Security
System limits, optimization, and threat models.
- **[Security Threat Model](infrastructure/security.md)**: STRIDE analysis and mitigations.
- **[Performance & Resource Limits](infrastructure/performance.md)**: Startup latency, async UI rendering, and conversation history summarization thresholds.

### 5. Operations & Releases
- **[Distribution & Build Matrix](operations/distribution.md)**: Release rules and cross-compilation configurations.
- **[Branching & Release Management](operations/versioning.md)**: Semantic versioning, branch naming, and release checklists.
- **[Maintainer Runbook](operations/runbooks.md)**: Project startup, troubleshooting local state, and error handling.

### 6. Roadmaps & History
- **[Project Roadmap](ROADMAP.md)**: Development phases, completed items, and future milestones.
- **Archive**:
  - **[Refactor Plan](archive/REFACTOR_PLAN.md)**: Historic refactor roadmap and planning.
  - **[Next Refactor Plan](archive/REFACTOR_PLAN_NEXT.md)**: Historic subsequent refactoring plans.

### 7. Standards & Reference
- **[TUI Design Standards](standard/tui-design.md)**: Spacing guidelines, color usage, spinner designs, and CLI layouts.
- **[Domain Glossary](standard/glossary.md)**: Vocabulary of system components and terms.
