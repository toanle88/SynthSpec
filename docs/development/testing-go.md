# Go Testing Strategies

This document describes how to write and execute Go tests in the SynthSpec repository.

## Unit Testing
- Store unit tests alongside source files using standard Go naming conventions (`*_test.go`).
- Focus unit tests on core state calculations, metrics evaluation, session serialization, source-first asset ordering, and fresh-prompt retry behavior.
- Mock external dependencies. Specifically, mock LLM gateways to prevent real API calls during tests.

### Running Unit Tests
Execute the unit tests using standard Go tools:
```bash
go test ./state/... ./generator/...
```

## Mocking the LLM Gateway
A `mock.go` client is provided in the `gateway/` package. Use this client for testing TUI workflows without requiring active API keys.
Ensure all tests using the gateway rely on mock payloads that validate:
1. Handling of different provider schemas.
2. Graceful simulation of HTTP 429 and rate-limiting responses.
3. Behavior of context summarization under simulated token constraints.
4. Fresh prompt retries for generation and refinement attempts.
