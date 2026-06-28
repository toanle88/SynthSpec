# SynthSpec Documentation Overview

Welcome to the SynthSpec project documentation. This workspace contains technical specifications, architecture diagrams, guidelines, and standards for developers and maintainers of the SynthSpec CLI tool.

- **[Product Requirements Document (PRD)](PRODUCT.md)**: Product requirements, functional requirements matrix, STRIDE threat modeling analysis, and output schemas.

## Directory Map

### 1. [Architecture Design](architecture/system.md)
Detailed breakdowns of the CLI's internal event-driven architecture, models, and component boundaries.
- **[System & Component Design](architecture/system.md)**: Entrypoint, State Controller, and overall architecture graph.
- **[TUI Dashboard Engine](architecture/tui.md)**: Asynchronous UI rendering, layouts, and input mapping.
- **[LLM Gateway](architecture/gateway.md)**: Multi-model routing, token optimization, and exponential backoff retry algorithms.

### 2. [Functional Specifications](spec/README.md)
Detailed product requirements and workflows mapped out.
- **[The Oracle Interrogation Loop](spec/interrogation_loop.md)**: The interactive single-question loop, confidence scores, and verification gates.
- **[Asset Generation (Source-First Synthesis)](spec/asset_generation.md)**: Source-doc-first generation, parallel downstream fan-out, output workspaces, and schemas.

### 3. [Standards & Reference](standard/glossary.md)
- **[TUI Design Standards](standard/tui-design.md)**: Spacing guidelines, color usage, spinner designs, and CLI layouts.
- **[Domain Glossary](standard/glossary.md)**: Vocabulary of system components and terms.

### 4. Development & Maintenance
- **[Go Coding Standards](development/coding-go.md)**: Directory structures, conventions, and style constraints.
- **[Testing Strategies](development/testing-go.md)**: Unit tests, mocking, and CLI integration tests.
- **[Security Threat Model](infrastructure/security.md)**: STRIDE analysis and mitigations.
- **[Distribution & Build Matrix](operations/distribution.md)**: Release rules and cross-compilation configurations.
- **[Maintainer Runbook](operations/runbooks.md)**: Project startup, troubleshooting local state, and error handling.
