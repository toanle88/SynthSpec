# ADR-004: Session Persistence Decoupling

## Status
Accepted

## Context
The generator pipeline originally held a direct reference to `*state.Session` and called `sess.Save()` internally. This created several problems:

1. **Testability**: Unit tests required real filesystem and session directory setup
2. **Layering violation**: `generator` (inner layer) depended on concrete `state.Session` (outer layer)
3. **Single responsibility**: Generator mixed synthesis logic with persistence concerns
4. **Coupling**: Changes to session structure required generator changes

## Decision
Introduce a `SessionPersistence` interface in `generator/persistence.go`:

```go
type SessionPersistence interface {
    SaveGeneratedFile(state GeneratedFileState) error
    LoadGeneratedFile(fileName string) (GeneratedFileState, bool)
    UpdateFacts(facts domain.Facts) error
    UpdateScores(scores domain.ConfidenceScores, rationales domain.DimensionRationales) error
    UpdateHistory(history []domain.Message) error
    UpdateTokens(prompt, completion int) error
    SaveSession() error
    GetProjectName() string
    GetProvider() string
    GetHistory() []domain.Message
    GetTotalTokens() int
    GetFacts() domain.Facts
}

type GeneratedFileState struct {
    FileName       string
    Results        []domain.ComplianceResult
    HasError       bool
    ErrMsg         string
    InProgressText string
    CurrentAttempt int
    PromptHash     string
    FactsHash      string
}
```

### Implementation
- `state.Session` implements `SessionPersistence` (in `state/session.go`)
- `generator.Generate()` accepts `SessionPersistence` interface
- `MockPersistence` in `generator/generator_test.go` for unit tests

### Injection Points
```go
// Before (coupled)
func Generate(ctx, gw, sess *state.Session, outputDir string, ...)

// After (decoupled)
func Generate(ctx, gw gateway.Gateway, persistence SessionPersistence, outputDir string, ...)
```

### Call Sites Updated
- `cmd/init.go` - passes `&sess` (implements interface)
- `cmd/resume.go` - passes `sess` 
- `cmd/update.go` - passes `sess`
- `tui/dashboard/commands.go` - passes `m.Session`
- `generator/generator_test.go` - passes `MockPersistence`

## Consequences

### Positive
- **Testability**: Generator tests run in-memory, no filesystem needed
- **Layering restored**: `generator` no longer imports `state`
- **Single responsibility**: Generator focuses on synthesis; persistence handled by `state`
- **Flexibility**: Can swap persistence backend (e.g., database, remote) without touching generator
- **Fast tests**: Unit tests complete in milliseconds

### Negative
- **Interface maintenance**: Adding persistence operations requires interface updates
- **Indirection**: Slight cognitive overhead tracing through interface

### Neutral
- `GeneratedFileState` duplicated in `generator` (avoids `generator → state` import)
- `state.Session` grows with interface methods (acceptable for concrete type)

## Related
- ADR-001: Layered Architecture
- ADR-003: Source-First Generation Pipeline