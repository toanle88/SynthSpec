# Branching & Release Management Rules

This document outlines the standard versioning and release procedures for the SynthSpec CLI tool.

## Versioning Standards
We use [Semantic Versioning](https://semver.org/) for release tags: `vMAJOR.MINOR.PATCH`.
- **MAJOR**: Breaking CLI command structures or file schemas (e.g. rewriting metadata config format).
- **MINOR**: New LLM provider integrations, additional spec output features, or major TUI dashboard enhancements.
- **PATCH**: Bug fixes, minor visual repairs, gateway stability corrections, and docs-only flow clarifications.

## Branching Model
- **`main`**: The primary branch. Only contains stable, tested release code. Direct pushes are blocked.
- **Feature Branches**: Create feature branches from `main` using naming conventions:
  - `feat/feature-description`
  - `fix/bug-description`
  - `docs/doc-description`

## Release Checklist
1. Verify all unit tests pass on local cross-platform scripts.
2. Increment the application version string in `cmd/root.go` or `PRODUCT.md`.
3. Update the changelog/milestone status in `docs/ROADMAP.md` and align the docs tree with any generation-flow changes.
4. Build target binaries for macOS, Linux, and Windows and verify their SHA-256 validation sums.
5. Create a GitHub Release with the tag `vX.Y.Z` containing the built binaries.
