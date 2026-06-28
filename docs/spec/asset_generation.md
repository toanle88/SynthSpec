# Asset Generation (Source-First Synthesis)

Once the interrogation loop reaches 100% confidence across all categories, **The Architect** — the expert AI Solution Engineer persona — unlocks the synthesis phase. This module generates a standardized project directory containing markdown documentation and machine-parsable configuration files.

The Architect executes a disciplined three-phase generation strategy:

1. **Source Phase** — Generates `01_domain_model_use_cases.md` first, then presents it for user approval at the **Domain Approval Gate** before proceeding.
2. **Parallel Phase** — Fans out the remaining 6 downstream documents concurrently, each undergoing independent synthesis, static validation, compliance auditing, and self-correction loops.
3. **Consolidation Phase** — Verifies cross-document logical consistency across all 7 deliverables, producing the final compliance report and metadata.

## Generation Strategy

Each generation or refinement attempt uses a fresh prompt. The approved `01_domain_model_use_cases.md` file is treated as the source of truth, and its final content is injected into downstream prompts as reference context.

## Output Directory Structure

The CLI outputs files into a target directory called `synthspec-output/`:

```plaintext
synthspec-output/
├── 00_compliance_report.md            # Standards compliance audit report
├── .synthspec-meta.json               # Execution details and provider stats
├── 01_domain_model_use_cases.md       # Source of truth for downstream docs
├── 02_prd_functional.md               # Product Requirements Document
├── 03_system_architecture.md          # Decoupled component design & schema layout
├── 04_api_architecture_integration.md  # API contract and transport rules
├── 05_coding_standards_guidelines.md   # Development standards and CI/CD guidance
├── 06_security_threat_model.md        # STRIDE threat modeling & mitigations
└── 07_engineering_roadmap.md          # Delivery phases and timeline planning
```

## Detailed File Specifications

### 1. Metadata Schema (`.synthspec-meta.json`)
Tracks internal execution details, project definitions, and foundational verification data.
See `PRODUCT.md` for the strict JSON Schema definition.

### 2. Source Document (`01_domain_model_use_cases.md`)
Establishes the domain model, bounded contexts, and scenario walkthroughs that anchor all downstream synthesis prompts.

### 3. Quality Audit (`00_compliance_report.md`)
Summarizes the standards evaluation results for generated assets and provides remediation details for failures.

### 4. API Contract (`04_api_architecture_integration.md`)
Captures API envelope rules, transport constraints, idempotency requirements, and validation behavior based on the domain source doc.
