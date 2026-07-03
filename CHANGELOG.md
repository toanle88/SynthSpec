# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2026-06-30] - Architecture Review Score 8.0

### Completed (from Refactor Plan)

- ✅ **#1: Extracted System Prompts** → `gateway/prompts.go` - eliminated prompt duplication across 4 provider files
- ✅ **#2: Moved `Standard` to `domain/`** - fixed layering violation; `gateway` no longer depends on `config`
- ✅ **#3: Removed `state → gateway` import** - `state/editor.go` now correctly uses `domain.Facts`
- ✅ **#4: Fixed silent error suppression** - added proper error logging in TUI and cmd paths
- ✅ **#5: Generic `loadYAML[T]` helper** → `config/yaml.go` - eliminated boilerplate in config loaders
- ✅ **#6: Split `config.go` into focused files** - `providers.go`, `standards.go`, `templates.go`, `blueprints.go`, `yaml.go`, `settings.go`
- ✅ **#7: Created `gateway/factory.go`** - centralized provider switch-case
- ✅ **#8: Extracted `resolveProjectName()` helper** → `cmd/helpers.go` - eliminated duplication in resume/update/export
- ✅ **#9: Added `Summarize()` to `Gateway` interface** - replaced `QueryOracle` misuse in `state/pruner.go`
- ✅ **#10: Decoupled generator from `*state.Session`** - introduced `SessionPersistence` interface in `generator/`, implemented in `state/`
- ✅ **#11: Expanded TUI state machine tests** - added tests for oracle result handling, generation flow, error paths
- ✅ **#12: Added hash computation unit tests** - `computeSha256` and `computeFactsHash` now have table-driven tests
- ✅ **#13: Derived file list from templates** - removed hardcoded `synthesisFiles` in `tui/dashboard/views_standards.go`
- ✅ **#14: Fixed `ExportMetadata.Version` hardcoding** - now uses `generator.EngineVersion`
- ✅ **#15: Removed empty directories** - deleted `generator/compliance/` and `generator/synthspec/`
- ✅ **#16: Fixed miscellaneous naming issues** - `TelemetryMetadata` → `GenerationMetadata`, `HasError/ErrorStr` → `ErrMsg`, etc.
- ✅ **#17: Evaluated `glamour` for markdown rendering** - determined incompatible with current Bubbletea/Lipgloss setup; added table-driven tests for existing renderer instead

### Added

- `gateway/prompts.go` - centralized system prompts
- `config/yaml.go` - generic YAML loading helper
- `config/providers.go`, `config/standards.go`, `config/templates.go`, `config/blueprints.go` - focused config files
- `gateway/factory.go` - provider factory
- `cmd/helpers.go` - shared command helpers
- `generator/persistence.go` - `SessionPersistence` interface
- `docs/architecture/decisions/` - Architecture Decision Records (ADRs)
- `docs/development/contributing.md` - contribution guidelines
- `docs/development/quickstart.md` - local development quickstart

### Changed

- `Standard` type moved from `config` to `domain` package
- `state/pruner.go` now uses `Gateway.Summarize()` instead of abusing `QueryOracle`
- Generator now uses `SessionPersistence` interface for testability
- TUI error handling now logs all errors instead of suppressing them
- Config package restructured into focused files

### Fixed

- Silent error suppression in `cmd/*.go` and `tui/dashboard/*.go`
- Import cycle risk between `state` and `gateway`
- Generator test isolation (no longer requires temp directories)

## [2026-06-29] - Architecture Review Score 7.4

### Added

- Initial project structure
- Multi-provider LLM gateway (Gemini, OpenAI, Anthropic, OpenRouter)
- Source-first generation pipeline with parallel fan-out
- Bubbletea-based TUI with dashboard and welcome screens
- Configuration system with embedded YAML and local overrides
- Session persistence with JSON serialization

### Changed

- N/A (initial release)

### Fixed

- N/A (initial release)