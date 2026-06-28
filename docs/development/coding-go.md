# Go Coding Standards

This document defines Go-specific coding standards and architectural patterns for the SynthSpec codebase.

## Code style and formatting
- **Formatting**: Run `go fmt` or `goimports` on all files before committing.
- **Linters**: Use `golangci-lint` to check code quality. Fix all warning reports before submitting PRs.
- **Naming Conventions**:
  - Follow standard Go naming (use camelCase, avoid underscores in variable names).
  - Keep interface names short, ending in `er` where applicable (e.g., `Provider`, `Generator`).
  - Package names must be lowercase, single-word nouns (e.g., `gateway`, `state`, `tui`).

## Package Structure
The directory layout follows standard Go CLI patterns:
- `cmd/`: CLI root commands (using `cobra` or standard flag packages). Handles parsing, flags, and command routing.
- `gateway/`: Abstract LLM provider logic. Isolates vendor payload models and endpoints.
- `state/`: Session management, configuration schemas, and file I/O.
- `tui/`: Asynchronous dashboard view model, keyboard event handlers, and styles.
- `generator/`: Source-first markdown output builder, parallel fan-out scheduler, and compliance report generation.

## Error Handling
- Never ignore errors. Handle them explicitly or return them wrapped.
- See [Error Handling Policies](errors.md) for more details.
