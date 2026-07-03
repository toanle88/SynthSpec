# ADR-002: Multi-Provider Gateway Interface

## Status
Accepted

## Context
SynthSpec needs to support multiple LLM providers (Gemini, OpenAI, Anthropic, OpenRouter) and a mock provider for testing. The system must allow users to choose their provider at runtime without code changes.

## Decision
Define a `Gateway` interface in `gateway/gateway.go` that all providers implement:

```go
type Gateway interface {
    QueryOracle(ctx context.Context, facts Facts, history []Message, latestInput string) (*OracleResponse, error)
    QueryOracleStream(ctx context.Context, facts Facts, history []Message, latestInput string, tokenChan chan<- string) (*OracleResponse, error)
    GenerateSpecFile(ctx context.Context, facts Facts, fileName string, promptTemplate string) (string, error)
    EvaluateCompliance(ctx context.Context, fileName string, fileContent string, standards []domain.Standard) ([]ComplianceResult, error)
    RefineSpecFile(ctx context.Context, fileName string, fileContent string, feedback string, failedStandards []domain.Standard, referenceDoc string) (string, error)
    VerifyConsistency(ctx context.Context, files map[string]string) (*ConsistencyReport, error)
    Summarize(ctx context.Context, history []Message) (string, error)
}
```

### Provider Implementations
- `gemini.go` - Google Gemini API
- `openai.go` - OpenAI API
- `anthropic.go` - Anthropic API
- `openrouter.go` - OpenRouter API (multi-model)
- `mock.go` - In-memory mock for testing

### Factory
`gateway/factory.go` provides `NewGateway(provider, apiKey, model string) (Gateway, error)` to centralize provider selection.

## Consequences

### Positive
- **Provider agnostic**: Core logic never knows which provider is used
- **Testable**: `MockGateway` enables fast, deterministic unit tests
- **Extensible**: Adding new providers only requires implementing the interface
- **Runtime switching**: Users can change providers via CLI flags

### Negative
- **Interface bloat**: All providers must implement all methods, even if some aren't used
- **Least common denominator**: Interface limited to capabilities all providers share

### Neutral
- Streaming (`QueryOracleStream`) adds complexity but enables real-time TUI updates

## Related
- ADR-001: Layered Architecture
- ADR-003: Source-First Generation Pipeline