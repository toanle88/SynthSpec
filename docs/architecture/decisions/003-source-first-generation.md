# ADR-003: Source-First Generation Pipeline

## Status
Accepted

## Context
The specification generation pipeline must produce 7 interdependent documents (domain model, PRD, architecture, API, coding standards, security, roadmap) with cross-document consistency. A naive parallel approach would produce inconsistent outputs.

## Decision
Implement a **source-first, parallel fan-out** pipeline in `generator/pipeline.go`:

### Phase 1: Source Document (Sequential)
1. Generate `01_domain_model_use_cases.md` first (the "source of truth")
2. **Approval gate**: Pause for human review/editing via TUI
3. Lock source document content

### Phase 2: Downstream Documents (Parallel)
4. Fan out to generate remaining 6 documents in parallel goroutines
5. Each downstream document receives the locked source document as reference
6. Self-correction loop per file: generate → validate → refine (up to 10 retries)

### Phase 3: Consistency Verification (Sequential)
7. Read all generated documents
8. Call `Gateway.VerifyConsistency()` for cross-document logical consistency
9. If inconsistent: refine offending documents (up to 3 loops)
10. Generate compliance report (`00_compliance_report.md`) and metadata (`.synthspec-meta.json`)

### Approval Gate Pattern
```go
type fileGenerator struct {
    approvalChan chan struct{}  // Closed when user approves in TUI
}

func (fg *fileGenerator) generateSourceDocument(...) {
    // ... generate source doc ...
    if fg.approvalChan != nil {
        sendProgress("waiting_approval", ...)
        select {
        case <-fg.approvalChan:  // User pressed 'a' in TUI
        case <-fg.ctx.Done():
            return ctx.Err()
        }
        // Re-read file to capture manual edits
    }
}
```

## Consequences

### Positive
- **Consistency**: Source document anchors all downstream generation
- **Human-in-the-loop**: Domain experts can correct misunderstandings early
- **Parallelism**: 6 downstream documents generated concurrently (~6x speedup)
- **Resilience**: Self-correction handles validation/compliance failures automatically
- **Auditability**: Metadata tracks tokens, turns, compliance scores

### Negative
- **Complexity**: Pipeline orchestration is non-trivial
- **Latency**: Approval gate blocks pipeline (mitigated by async TUI)
- **Resource usage**: Parallel generation consumes more API quota simultaneously

### Neutral
- Requires `approvalChan` plumbing through TUI → generator

## Related
- ADR-001: Layered Architecture
- ADR-002: Gateway Interface
- ADR-004: Session Persistence Decoupling