# SynthSpec Refactoring Plan — Next Phase (Post 8.0 Review)

> Based on Architecture Review (2026-06-30) | Current Score: **8.0/10** | Target: **8.5+**

---

## Executive Summary

The architecture review confirms **significant progress** from 7.4 → 8.0. The remaining gaps are concentrated in:
1. **Generator ↔ State coupling** (Refactor #10) — highest architectural leverage
2. **Missing `Summarize()` on Gateway** (Refactor #9) — semantic clarity
3. **Thin TUI test coverage** (Refactor #11) — regression risk
4. **Housekeeping** — `.gitignore`, binary cleanup, stale docs

---

## Priority 1 — Immediate (Low Effort, High Impact)

### 1.1 Add `synthspec/` and `*.exe` to `.gitignore` ✅ **VERIFIED DONE**
- `.gitignore` already contains:
  ```
  synthspec/
  synthspec.exe
  *.exe
  ```
- **Action**: Verify `synthspec.exe` is not tracked (`git status` should show it as ignored)

### 1.2 Fix Remaining Silent Error Suppressions

| File | Line | Current | Fix |
|------|------|---------|-----|
| `tui/dashboard/model.go` | ~147, ~176 | `standards, _ := config.LoadStandards()` | Add error logging |
| `tui/dashboard/handlers.go` | ~79, ~101, ~195 | `m.Session.Save()` unchecked | Check and log error |
| `cmd/root.go`, `cmd/init.go`, `cmd/resume.go`, `cmd/update.go` | various | `settings, _ := config.LoadSettings()` | Add error logging |

**Pattern to apply:**
```go
standards, err := config.LoadStandards()
if err != nil {
    logger.Log("WARN: failed to load standards: %v", err)
}
```

### 1.3 Archive/Retire `REFACTOR_PLAN.md`
- Move completed items → `CHANGELOG.md` (create if missing)
- Move remaining items → GitHub Issues or `docs/development/refactor-history.md`
- Remove `REFACTOR_PLAN.md` from repo root

---

## Priority 2 — Short-Term (Medium Effort, High Architectural Value)

### 2.1 Add `Summarize()` to `Gateway` Interface (Refactor #9)

**Files to modify:**
- `gateway/gateway.go` — add method signature
- `gateway/gemini.go`, `openai.go`, `anthropic.go`, `openrouter.go`, `mock.go` — implement
- `state/pruner.go` — replace `QueryOracle` misuse with `Summarize()`

**Interface addition:**
```go
// Summarize generates a concise summary of the conversation history
Summarize(ctx context.Context, history []Message) (string, error)
```

**Implementation notes:**
- Each provider should use a focused summarization prompt (not the Oracle prompt)
- Mock implementation can return a simple concatenation or fixed string for tests
- `state/pruner.go` currently calls `gw.QueryOracle()` and extracts `NextQuestion` — replace with `gw.Summarize()`

**Validation:** `go build ./... && go test ./state/... ./gateway/...`

---

### 2.2 Decouple Generator from `*state.Session` (Refactor #10) — **HIGHEST LEVERAGE**

**Problem:** `generator/fileGenerator` holds `sess *state.Session` and calls `sess.Save()` internally. This couples synthesis logic to disk persistence, making unit tests require a real filesystem.

**Solution:** Define a `SessionPersistence` interface in `generator/` and inject it.

**Files to create/modify:**

1. **Create `generator/persistence.go`** (new file):
```go
package generator

import "github.com/toanle/synthspec/domain"

type SessionPersistence interface {
    SaveGeneratedFile(state GeneratedFileState) error
    LoadGeneratedFile(fileName string) (GeneratedFileState, bool)
    SaveSession() error
}

// GeneratedFileState mirrors state.GeneratedFileState but lives in generator
// to avoid generator → state dependency
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

2. **Modify `generator/generator.go`**:
   - Change `fileGenerator.sess *state.Session` → `persist SessionPersistence`
   - Update `generateSourceDocument`, `generateDownstreamParallel`, etc. to use `persist.SaveGeneratedFile()`
   - Remove direct `sess.Save()` calls

3. **Implement in `state/session.go`**:
```go
func (s *Session) SaveGeneratedFile(state generator.GeneratedFileState) error {
    // Convert and append to s.GeneratedFiles
    return s.Save()
}

func (s *Session) LoadGeneratedFile(fileName string) (generator.GeneratedFileState, bool) {
    // Search s.GeneratedFiles
}

func (s *Session) SaveSession() error {
    return s.Save()
}
```

4. **Update call sites** (`cmd/init.go`, `cmd/resume.go`, `cmd/update.go`) to pass `sess` as `SessionPersistence`

**Validation:** `go build ./... && go test ./generator/...` — tests should no longer need temp directories for session persistence

---

### 2.3 Add `CONTRIBUTING.md` and `CHANGELOG.md`

**Create `docs/development/contributing.md`:**
- Development setup (Go 1.26+, `go mod tidy`, `go build`)
- Code style (gofmt, golangci-lint)
- Testing expectations (`go test ./...`)
- PR process, commit message conventions
- Architecture overview reference (`docs/architecture/system.md`)

**Create `CHANGELOG.md` at root:**
- Move completed refactor items from `REFACTOR_PLAN.md` here
- Use Keep a Changelog format (Added/Changed/Fixed/Removed)

---

## Priority 3 — Medium-Term (Test Coverage & Documentation)

### 3.1 Expand TUI State Machine Tests (Refactor #11)

**Current state:** `tui/dashboard/model_test.go` (2411 bytes) + `update_test.go` (1445 bytes) — mostly construction smoke tests.

**Target flows to test:**
| Flow | Test Approach |
|------|---------------|
| Oracle result received → session state updated | Call `Update()` with synthesized `OracleResultMsg`, assert `m.Session.Facts` updated |
| Generation started → `m.isGenerating = true` | Send `GenerationStartedMsg`, assert state |
| Generation finished → status updated | Send `GenerationCompletedMsg`, assert `m.genStatus` |
| Error handling paths | Send `ErrorMsg`, assert error display state |
| Approval gate transitions | Simulate approval/rejection messages |

**Pattern to follow:**
```go
func TestDashboardModel_OracleResultReceived(t *testing.T) {
    sess := &state.Session{...}
    gw := gateway.NewMockGateway()
    m := NewDashboardModel(sess, gw, "")
    
    // Simulate oracle result message
    msg := OracleResultMsg{Response: &gateway.OracleResponse{...}}
    m.Update(msg)
    
    // Assert state transitions
    if !m.isGenerating {
        t.Error("expected isGenerating=true after oracle result")
    }
}
```

**Files:** `tui/dashboard/model_test.go`, `tui/dashboard/update_test.go`

---

### 3.2 Add Architecture Decision Records (ADRs)

**Create `docs/architecture/decisions/`** with:
- `001-clean-architecture-layering.md` — domain → gateway → generator → tui
- `002-multi-provider-gateway-abstraction.md` — Gateway interface pattern
- `003-source-first-generation-pipeline.md` — pipeline.go design
- `004-session-persistence-decoupling.md` — Refactor #10 rationale

**Template:**
```markdown
# ADR-NNN: Title

## Status
Accepted / Proposed / Superseded

## Context
What problem are we solving?

## Decision
What did we decide?

## Consequences
Trade-offs, follow-up work
```

---

### 3.3 Local Dev Quickstart Doc

**Create `docs/development/quickstart.md`:**
- Prerequisites (Go 1.26+, API keys)
- `go mod tidy && go build -o synthspec.exe main.go`
- Run TUI: `./synthspec.exe`
- Run tests: `go test ./...`
- Project structure overview (2-min read)

---

## Priority 4 — Polish (Low Priority)

| Item | Refactor # | Effort | Notes |
|------|------------|--------|-------|
| Derive file list from templates (not hardcoded) | #13 | Low | `tui/dashboard/views_standards.go` |
| Fix `ExportMetadata.Version` hardcoding | #14 | Trivial | `generator/export/exporter.go:119` |
| Remove empty dirs `generator/compliance/`, `generator/synthspec/` | #15 | Trivial | `rm -rf` |
| Naming fixes (TelemetryMetadata→GenerationMetadata, etc.) | #16 | Low | See REFACTOR_PLAN.md table |
| Evaluate `glamour` for markdown rendering | #17 | Medium | Spike first, then decide |

---

## Execution Order & Validation Gates

```
Phase 1 (Immediate)          → go build ./... && git status (clean)
    ├── 1.1 Verify .gitignore ✓
    ├── 1.2 Fix silent errors
    └── 1.3 Archive REFACTOR_PLAN.md

Phase 2 (Short-term)         → go build ./... && go test ./...
    ├── 2.1 Add Summarize() to Gateway
    ├── 2.2 Decouple generator from *state.Session  ← BIGGEST WIN
    └── 2.3 Add CONTRIBUTING.md + CHANGELOG.md

Phase 3 (Medium-term)        → go test ./tui/... (coverage ↑)
    ├── 3.1 TUI state machine tests
    ├── 3.2 ADRs
    └── 3.3 Quickstart doc

Phase 4 (Polish)             → go build ./... && go vet ./...
    ├── 4.1-4.5 Incremental cleanup
```

---

## Success Criteria for 8.5+

| Metric | Current | Target |
|--------|---------|--------|
| Generator test isolation | Requires temp dir + session | Pure unit tests, no FS |
| Gateway interface completeness | Missing `Summarize()` | Complete |
| TUI state transition coverage | ~15% | >60% |
| Silent error suppressions | ~6 locations | 0 |
| Stale docs at root | REFACTOR_PLAN.md | CHANGELOG.md + ADRs |
| Binary in repo | synthspec.exe (ignored) | Not tracked |

---

## Notes for Implementers

1. **Phase 2.2 (Generator decoupling)** is the single highest-leverage change. It enables fast, isolated unit tests for the generation pipeline — the most complex logic in the codebase.

2. **Phase 2.1 (Summarize)** is a prerequisite for clean pruner logic and should be done first (it's smaller).

3. **Run `go test ./...` after each phase** — the test suite is the safety net.

4. **Update `docs/architecture/system.md`** if layer boundaries change during Phase 2.

5. **Consider opening GitHub Issues** for Phase 3+ items to track them visibly rather than in a local markdown file.