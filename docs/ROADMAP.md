# Project Roadmap & Milestones

This document tracks current priorities, upcoming milestones, and development status for the SynthSpec CLI tool.

## Completed Milestones

### Milestone 1: Core CLI & LLM Gateway (v0.1.0) ── [x] Completed
- [x] Abstract Gateway interface supporting multiple providers (Gemini, OpenAI, Anthropic, OpenRouter).
- [x] Environment variable authentication logic (`GEMINI_API_KEY`, `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, etc.).
- [x] Initial Go CLI setup with commands `init` and `resume`.

### Milestone 2: Asynchronous TUI & State Controller (v0.2.0) ── [x] Completed
- [x] Multi-panel layout configuration (Header, Metrics, Active Question, User Input).
- [x] State persistence serializer saving to local `.synthspec/session.json`.
- [x] Score calculation mechanics evaluating categorical completion confidence.
- [x] Non-blocking TUI spinner during LLM API round-trips.

### Milestone 3: Interrogation Mechanics (v0.3.0) ── [x] Completed
- [x] Implement strict Single Question Constraint on LLM Prompts.
- [x] Link `:edit` command to fork host editor (`$EDITOR`) to edit transient context.
- [x] Background thread for Context Pruning & summarization when exceeding 75% context limit.

### Milestone 4: Asset Generation (Source-First Synthesis) (v0.4.0) ── [x] Completed
- [x] Enable the asset generator gate to unlock at 100% confidence.
- [x] Build metadata `.synthspec-meta.json` schema compiler.
- [x] Generate `01_domain_model_use_cases.md` first and fan out downstream documents in parallel.
- [x] Produce the compliance audit report and validate generated outputs against target standards.

---

## Upcoming Milestones (Iterative Hardening & Scaling)

### Milestone 5: Interactive Boot Menu & Project Manager (v0.5.0) ── [x] Completed
- [x] **Welcome Screen**: Implement an interactive selection menu on boot using a TUI component library, removing the need to memorize CLI subcommands.
- [x] **Industry-Specific Blueprints**: Introduce starting templates (e.g., `fintech-saas`, `internal-crud`) during project initialization to pre-load context and accelerate the interrogation loop.
- [x] **Action Routing & Global Settings**: Provide global menu routes including "Create New Project", "Resume Existing", "View Assets", **"Audit Workspace (Drift Detection)"**, and a "Settings" configuration pane. 
- [x] **Configurable Application Parameters**: Allow users via the Settings pane to customize network constraints (API timeout limits and max retries) and set a default output folder for generated specifications.
- [x] **Ephemeral Debug Logging**: Add an opt-in `--debug` flag to write sanitized execution traces to `.synthspec/crash.log` for troubleshooting without violating the zero-data retention policy.
- [x] **Fuzzy Project Finder**: Build a dynamic directory scanner that populates a searchable, fuzzy-filtered list of existing projects.

### Milestone 6: In-App Document Viewer & UX Hardening (v0.6.0) ── [ ] Planned
- [x] Upgrade the "Document Synthesis Status" panel to support interactive file selection using keyboard arrows.
- [x] Implement an integrated split-pane or full-screen markdown viewer with ANSI syntax highlighting.
- [x] **Vim & Accessibility Keybindings**: Add a configuration toggle for Vim-style (hjkl) navigation and ensure TUI interactions support standard accessible key maps.
- [x] **Streaming Token Visualization**: Upgrade the static TUI spinner to a "Streaming Thought Box" that exposes raw reasoning tokens in a dimmed secondary panel to confirm responsiveness during long queries.
- [ ] **The "I Don't Know" Fallback**: Map a hotkey (e.g., `Ctrl+K`) allowing users to request AI recommendations for highly specific compliance or architecture questions during the interrogation loop.
- [ ] **The Domain Approval Gate**: Enforce a hard pause after generating `01_domain_model_use_cases.md`, presenting it in the viewer for explicit user sign-off (and editing) before unlocking the parallel generation of downstream documents.
- [ ] **Static Site Generator (SSG) Export**: Add a compilation pipeline to build the generated Markdown workspace into a standalone, searchable static HTML site (e.g., via Docusaurus or MkDocs templates) for presentation to non-technical stakeholders.
- [ ] Extend HTTP client timeouts and implement robust retry/error handling to resolve `context deadline exceeded` errors.
- [ ] Enable full mouse interaction in the TUI (clicking panels, scrolling lists) using the underlying terminal framework.

### Milestone 7: The Agentic Guardrail Compiler (v0.7.0) ── [ ] Planned
- [ ] **Context Injection Engine**: Synthesize lightweight pointer files (`.cursor/rules/*.mdc`, `CLAUDE.md`, or `AGENTS.md`) that explicitly link down to the generated markdown specifications.
- [ ] **Structural Entity Extraction**: Build a background parser that converts the human-readable `01_domain_model_use_cases.md` into dense, optimized YAML/JSON data objects before feeding it into downstream parallel prompts to minimize token tax and prevent "lost in the middle" hallucination.
- [ ] **Prompt Optimization**: Implement a backend routine to condense human-readable markdown files into absolute, imperative directives optimized for downstream coding LLMs.
- [ ] **Live Token & Cost Estimation**: Track exact tokens consumed during the session and display a real-time estimated cost/burn-rate metric in the TUI header.
- [ ] **Circuit Breakers & Hard Budgets**: Build a safety monitoring mechanism that allows users to configure a hard monetary cap (e.g., max $2.00 per execution) to automatically halt background API routines if a loop begins consuming excessive tokens.
- [ ] **Pre-Flight Secret Scrubbing**: Integrate a local regex scanner (e.g., gitleaks logic) to intercept the prompt and block requests if a user accidentally pastes proprietary secrets (API keys, passwords) into the TUI.

### Milestone 8: Smart Diffing & Session State (v0.8.0) ── [ ] Planned
- [ ] Implement `synthspec update` command (tied to the Boot Menu) to load existing `.synthspec-meta.json` and generated markdown files.
- [ ] **Internal Consistency Auditor (Docs vs. Docs)**: Implement a verification sweep that cross-references generated JSON/YAML structural entities across all Markdown files to highlight manual editing discrepancies (e.g., flagging if an entity exists in the API spec but is missing from the Domain Model).
- [ ] Build a structured diffing engine to calculate delta changes based on newly added requirements without overwriting manual developer edits.
- [ ] **Colorblind-Safe Diffing**: Ensure the interactive Git-style diff viewer utilizes explicit ANSI symbols (+ / -) or high-contrast themes instead of relying solely on red/green color coding.
- [ ] **Self-Healing Markdown Linting**: Add a post-generation validation step to parse and verify structural integrity (e.g., tables, code blocks). Automatically trigger a silent LLM repair loop for malformed syntax.
- [ ] **Bi-Directional Architecture Updates**: Implement a retroactive feedback loop that detects when a downstream document (like API Architecture) identifies a required edge-case entity, and intelligently proposes an upstream update back to the root Domain Model without overwriting the entire file.
- [ ] **Human-in-the-Loop (HITL) Confidence Override**: Introduce an escape hatch command (`:override` or `:bypass`) allowing engineers to manually fast-track lagging evaluation segments to 100% confidence, preventing deadlocks caused by pedantic LLM sub-queries.
- [ ] Add an interactive Git-style diff viewer in the TUI for user approval before modifying physical files.
- [ ] **Session "Time-Travel"**: Implement a state-history stack allowing the user to type `:undo` or press `Ctrl+Z` to step backward in the conversation.

### Milestone 9: Context Ingestion & Custom Standards Blueprint (v0.9.0) ── [ ] Planned
- [ ] Add `synthspec ingest <path>` command to index existing codebases, database schemas, and legacy documentation via local embeddings (RAG).
- [ ] **Standards & Prompt Overrides (BYOT)**: Allow execution customization by exposing standard definitions (`standards.yaml`) and template instructions (`templates.yaml`) to local project folder deep-merges, allowing teams to redefine engineering criteria.
- [ ] **Rich Diagram Export**: Add a synthesis pipeline to output structural diagrams directly into `.excalidraw` JSON format and Structurizr `.dsl` (C4 Model) files for immediate visual editing.

---

## Long-Term Vision (Enterprise Governance & Autonomy)

### Milestone 10: Executable Architecture & Scaffolding (v1.0.0) ── [ ] Planned
- [ ] **Zero-to-Code Scaffolding Engine**: Add a `synthspec scaffold` command to automatically generate the physical code directory topography, base interfaces, and domain structs in the target language based on the defined architecture.
- [ ] **Issue Tracker Synchronization**: Build an automated handoff utility (`synthspec sync`) to parse functional requirements from generated documents and push them directly to Jira, Linear, or GitHub Issues as formatted epics and sub-tasks.
- [ ] **Mock Server Generation**: Programmatically construct containerized, local mock environments (via WireMock or Prism configuration bundles) mirroring the synthesized API architecture maps, allowing front-end validation to begin instantly.
- [ ] **Executable Behavioral Specifications**: Translate functional requirements from the PRD into BDD Feature Files (Cucumber/Gherkin) and TDD preconditions.
- [ ] **Infrastructure as Code (IaC) Synthesis**: Generate production-ready Terraform or Pulumi configurations detailing VPCs, IAM roles, and RDS instances required by the system's threat model.

### Milestone 11: Production-Grade Enforcement (v1.1.0) ── [ ] Planned
- [ ] **Architecture Drift Detection (Docs vs. Code)**: Synthesize automated architecture tests using frameworks like ArchUnit or NetArchTest, and expose an on-demand audit route in the Boot Menu to map physical local code against the established `.synthspec-meta.json` constraints to flag violations.
- [ ] **SynthSpec LSP (Language Server Protocol)**: Build a local language server that integrates into VS Code/Neovim to provide real-time editor warnings (red squiggles) the moment a developer types code that violates the generated architectural boundaries.
- [ ] **Persistent Organizational Memory**: Build a global `~/.synthspec/` knowledge graph that learns team preferences across projects to accelerate future initializations.

---

## Next-Gen Horizons (Active Ecosystem Integration)

### Milestone 12: Reverse Engineering & Simulation (v2.0.0+) ── [ ] Planned
- [ ] **The "Brownfield" Engine**: Implement a legacy ingestion command (`synthspec audit <path>`) to scan, map, and reverse-engineer existing undocumented codebases back into structured specifications and technical debt reports.
- [ ] **Dynamic Context Engineering (Anti-Rot)**: Generate task-specific context pointers that curate only the minimum high-signal tokens needed for a specific coding task, combating "context rot" and the "lost in the middle" phenomenon.
- [ ] **Multi-Agent Orchestration Protocols**: Generate coordination configurations designed to orchestrate entire teams of specialized AI sub-agents working in parallel.
- [ ] **What-If Architecture Simulation**: Build an execution sandbox to safely simulate architectural shifts and dynamically project cascading design changes along with live infrastructure cost alterations via cloud pricing APIs.
- [ ] **Chaos Engineering & Threat Simulation**: Convert threat modeling documents directly into actionable verification scripts to programmatically audit deployed environments against the established spec.

### Milestone 13: Enterprise Air-Gapped & Collaborative Intelligence (v3.0.0+) ── [ ] Planned
- [ ] **The Air-Gapped Engine (Local Inference)**: Build native adapter support for Ollama and Llama.cpp, allowing the entire AI architecture generation process to run on local, offline compute clusters for maximum privacy in highly regulated environments.
- [ ] **Multi-Player Collaborative TUI**: Integrate an embedded terminal SSH server (e.g., via Charm `wish`) allowing multiple developers to securely join a shared remote SynthSpec session, enabling team-based real-time architectural decision-making.
- [ ] **Continuous RLHF Learning Loop**: Automatically monitor and capture manual human modifications (diffs) made to generated architectures, compiling a structured `.jsonl` preference dataset used to automatically fine-tune custom AI adapters that learn and mimic an organization's bespoke engineering culture.