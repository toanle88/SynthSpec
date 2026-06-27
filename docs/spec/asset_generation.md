# Asset Generation (The Draftsman)

Once the interrogation loop reaches 100% confidence across all categories, SynthSpec unlocks the synthesis phase (code-named "The Draftsman"). This module generates a standardized project directory containing markdown documentation and machine-parsable configuration files.

## Output Directory Structure

The CLI outputs files into a target directory called `synthspec-output/`:

```
synthspec-output/
├── .synthspec-meta.json       # Execution details and provider stats
├── 01_prd_functional.md       # Product Requirements Document
├── 02_system_architecture.md   # Decoupled component design & schema layout
├── 03_security_threat_model.md # STRIDE threat modeling & mitigations
├── 04_openapi_contract.yaml    # Synthesized REST/gRPC API definitions
└── 05_engineering_backlog.json # Tasks structured as Epics & User Stories
```

## Detailed File Specifications

### 1. Metadata Schema (`.synthspec-meta.json`)
Tracks internal execution details, project definitions, and foundational verification data.
See `PRODUCT.md` for the strict JSON Schema definition.

### 2. Engineering Backlog Schema (`05_engineering_backlog.json`)
Structures functional units into a clean task schema suitable for direct ingestion into project management tools like Jira, Linear, or GitHub issues.
See `PRODUCT.md` for the strict JSON Schema definition.

### 3. API Contract (`04_openapi_contract.yaml`)
Synthesizes standard OpenAPI 3.0 YAML declarations mapping out endpoints, query parameters, request payloads, and response objects based on the structural requirements established during the interrogation loop.
