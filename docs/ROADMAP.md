# Project Roadmap & Milestones

This document tracks current priorities, upcoming milestones, and development status for the SynthSpec CLI tool.

## Milestones

### Milestone 1: Core CLI & LLM Gateway (v0.1.0) ── [x] Completed
- [x] Abstract Gateway interface supporting multiple providers (Gemini, OpenAI, Anthropic).
- [x] Environment variable authentication logic (`GEMINI_API_KEY`, `OPENAI_API_KEY`, etc.).
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

### Milestone 4: Asset Generation (The Draftsman) (v0.4.0) ── [ ] Planned
- [ ] Enable the asset generator gate to unlock at 100% confidence.
- [ ] Build metadata `.synthspec-meta.json` schema compiler.
- [ ] Export system diagrams, OpenAPI specifications (`04_openapi_contract.yaml`), and backlogs (`05_engineering_backlog.json`).
- [ ] Validate generated backlogs against target schemas.
