# Contributing to SynthSpec

Thank you for your interest in contributing to SynthSpec! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Architecture Principles](#architecture-principles)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Commit Message Convention](#commit-message-convention)

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/synthspec.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`
6. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.26+
- Git
- An API key for at least one LLM provider (Gemini, OpenAI, Anthropic, or OpenRouter) for integration testing

### Building

```bash
# Build the binary
go build -o synthspec.exe main.go

# Or use the Makefile if available
make build
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./generator/... -v
go test ./tui/dashboard/... -v

# Run a specific test
go test ./generator/... -run TestGenerate_AllSuccess -v
```

### Running the Application

```bash
# Initialize a new project
./synthspec.exe init my-project --provider gemini --api-key $GEMINI_KEY

# Resume an existing project
./synthspec.exe resume my-project

# List all projects
./synthspec.exe list

# Export to HTML
./synthspec.exe export my-project
```

## Project Structure

```
synthspec/
├── cmd/                    # Cobra CLI commands
│   ├── init.go            # Initialize new project
│   ├── resume.go          # Resume existing project
│   ├── update.go          # Update project requirements
│   ├── export.go          # Export to HTML
│   ├── list.go            # List projects
│   ├── delete.go          # Delete project
│   ├── root.go            # Root command & welcome TUI
│   └── helpers.go         # Shared command helpers
├── config/                # Configuration loading
│   ├── providers.go       # Provider constants & detection
│   ├── standards.go       # Quality standards loading
│   ├── templates.go       # Document templates loading
│   ├── blueprints.go      # Blueprint loading
│   ├── settings.go        # User settings
│   └── yaml.go            # Generic YAML loader
├── domain/                # Pure domain types (no external deps)
│   └── domain.go          # Core types: Standard, Facts, Message, etc.
├── gateway/               # LLM provider abstraction
│   ├── gateway.go         # Gateway interface
│   ├── factory.go         # Provider factory
│   ├── prompts.go         # System prompts
│   ├── gemini.go          # Google Gemini implementation
│   ├── openai.go          # OpenAI implementation
│   ├── anthropic.go       # Anthropic implementation
│   ├── openrouter.go      # OpenRouter implementation
│   └── mock.go            # Mock for testing
├── generator/             # Specification generation pipeline
│   ├── generator.go       # Main entry point
│   ├── pipeline.go        # Source-first + parallel generation
│   ├── file_processor.go  # Per-file generation with self-correction
│   ├── session_state.go   # Session state management
│   ├── persistence.go     # SessionPersistence interface
│   ├── compliance.go      # Compliance evaluation
│   ├── validators.go      # Static validation
│   ├── backlog.go         # Engineering backlog generation
│   └── export/            # HTML export functionality
├── state/                 # Session persistence
│   ├── session.go         # Session struct & persistence
│   ├── pruner.go          # Context window management
│   ├── editor.go          # External editor integration
│   └── limits.go          # Model token limits
├── tui/                   # Terminal UI (Bubbletea)
│   ├── dashboard/         # Main interrogation/generation dashboard
│   │   ├── model.go       # Dashboard model & state
│   │   ├── update.go      # Update handlers
│   │   ├── commands.go    # Background commands
│   │   ├── handlers.go    # Message handlers
│   │   ├── views*.go      # View rendering
│   │   └── keys/          # Key bindings
│   ├── welcome/           # Welcome/startup screen
│   └── shared/            # Shared TUI utilities
├── logger/                # Logging utilities
├── shared/                # Shared utilities
├── docs/                  # Documentation
│   ├── architecture/      # Architecture docs & ADRs
│   ├── development/       # Development guides
│   ├── spec/              # Specification docs
│   ├── standard/          # Standards & glossary
│   ├── infrastructure/    # Performance & security
│   └── operations/        # Runbooks & versioning
└── testdata/              # Test fixtures
```

## Architecture Principles

### Layer Separation (Clean Architecture)

```
domain → gateway → generator → tui → cmd
```

- **domain**: Pure Go types, zero external dependencies
- **gateway**: LLM provider abstraction, depends only on domain
- **generator**: Generation pipeline, depends on gateway + domain
- **tui**: Terminal UI, composes everything
- **cmd**: CLI entry points, wires everything together

### Dependency Rules

1. Inner layers never depend on outer layers
2. Interfaces defined in inner layers, implemented in outer layers
3. Use dependency injection for cross-layer communication

### Key Patterns

- **Gateway Interface**: All LLM providers implement `gateway.Gateway`
- **SessionPersistence**: Generator uses interface for session persistence (testability)
- **Source-First Pipeline**: Generate source document first, then parallel fan-out
- **Self-Correction Loop**: Syntax validation → compliance evaluation → refinement

## Code Style

### Go Standards

- Follow standard Go formatting: `gofmt` / `goimports`
- Use `golangci-lint` for static analysis
- Prefer explicit errors over panics
- Use meaningful variable names (avoid single-letter except in tight loops)
- Document exported types and functions

### Running Linters

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...
```

### Import Organization

```go
import (
    // Standard library
    "context"
    "fmt"
    
    // Third-party
    "github.com/charmbracelet/bubbletea"
    
    // Local
    "github.com/toanle/synthspec/config"
    "github.com/toanle/synthspec/domain"
)
```

## Testing

### Test Organization

- Unit tests: `*_test.go` alongside source files
- Integration tests: Use `testdata/` fixtures
- Mock implementations: In same package (e.g., `gateway/mock.go`)

### Test Guidelines

1. **Test behavior, not implementation** - test state transitions, not internal fields
2. **Use table-driven tests** for multiple input/output cases
3. **Mock external dependencies** - use interfaces for testability
4. **Keep tests fast** - avoid real API calls in unit tests

### Example Test Pattern

```go
func TestGenerate_AllSuccess(t *testing.T) {
    tempDir := t.TempDir()
    persistence := NewMockPersistence()
    tg := &TestGateway{...}
    
    err := Generate(context.Background(), tg, persistence, tempDir, progress, nil)
    if err != nil {
        t.Fatalf("expected success, got err: %v", err)
    }
    // Assert file existence, call counts, etc.
}
```

## Pull Request Process

1. **Create a focused PR** - one logical change per PR
2. **Update documentation** - if you change behavior, update relevant docs
3. **Add tests** - new functionality should have test coverage
4. **Run full test suite** - `go test ./...` must pass
5. **Run linter** - `golangci-lint run ./...` must pass
6. **Update CHANGELOG.md** - add entry under `[Unreleased]`

### PR Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run ./...`)
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] No breaking changes without discussion
- [ ] Commit messages follow convention

## Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Performance improvement
- `test`: Adding missing tests
- `chore`: Maintenance, build process, etc.

### Examples

```
feat(generator): add parallel generation for downstream documents

fix(gateway): handle empty response from Anthropic API

docs(architecture): add ADR for source-first pipeline

refactor(config): split config.go into focused files

test(generator): add table-driven tests for computeSha256
```

### Scope Guidelines

Use the package name as scope when applicable:
- `cmd/`, `config/`, `domain/`, `gateway/`, `generator/`, `state/`, `tui/`, `logger/`, `shared/`

---

## Questions?

- Check existing [issues](https://github.com/toanle/synthspec/issues)
- Review [architecture docs](docs/architecture/)
- Open a new issue for discussion

Thank you for contributing! 🚀