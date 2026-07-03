# ADR-001: Layered Architecture

## Status
Accepted

## Context
SynthSpec is a CLI application that transforms vague application ideas into production-ready engineering specifications using LLMs. The codebase needs clear separation of concerns to maintain testability, allow provider swapping, and enable independent evolution of components.

## Decision
We adopt a strict layered architecture with the following dependency flow:

```
domain → gateway → generator → tui → cmd
```

### Layer Responsibilities

| Layer | Package | Responsibility | Dependencies |
|-------|---------|----------------|--------------|
| Domain | `domain/` | Pure data types, zero external deps | None |
| Gateway | `gateway/` | LLM provider abstraction | `domain` |
| Generator | `generator/` | Specification synthesis pipeline | `gateway`, `domain`, `config` |
| TUI | `tui/` | Terminal user interface | `generator`, `gateway`, `state`, `config` |
| CLI | `cmd/` | Command-line entry points | All above |

### Rules

1. **Inner layers never import outer layers** - `domain` has zero imports from the project
2. **Interfaces defined in consumer layer** - `generator` defines `SessionPersistence` interface; `state` implements it
3. **Dependency injection** - `generator.Generate()` accepts `SessionPersistence` interface, not concrete `*state.Session`
4. **Factory pattern** - `gateway.NewGateway()` centralizes provider instantiation

## Consequences

### Positive
- **Testability**: Each layer can be tested in isolation with mocks
- **Swappability**: LLM providers are interchangeable via `Gateway` interface
- **Maintainability**: Changes to one layer don't cascade to others
- **Parallel development**: Teams can work on different layers independently

### Negative
- **Boilerplate**: Interface definitions and DI add some verbosity
- **Indirection**: Debugging requires tracing through interfaces

### Neutral
- Requires discipline to maintain boundaries during code reviews

## Related
- ADR-002: Gateway Interface
- ADR-003: Source-First Generation Pipeline
- ADR-004: Session Persistence Decoupling