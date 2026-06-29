# SynthSpec Refactoring Plan

> Based on architecture review (2026-06-29) | Score: 7.4 → target: 8.5+

---

## Phase 1 — Quick Wins (Low effort, High impact)

### 🔴 #1: Extract System Prompts → `gateway/prompts.go`

**What:** System prompt strings (~40 lines for QueryOracle, ~15 for EvaluateCompliance, etc.) are copy-pasted verbatim across all 4 provider files (`gemini.go`, `openai.go`, `anthropic.go`, `openrouter.go`).

**Steps:**

1. **Create** `gateway/prompts.go` with exported constants:
   - `OracleSystemPrompt` — the ~40-line system prompt for `QueryOracle`/`QueryOracleStream`
   - `ComplianceSystemPrompt` — the ~15-line prompt for `EvaluateCompliance`
   - `RefinementSystemPrompt` — the ~3-line prompt for `RefineSpecFile`
   - `ConsistencySystemPrompt` — the ~10-line prompt for `VerifyConsistency`

2. **Verify exact prompt text** — read the raw string literal from each provider to ensure they are truly identical. If there are minor variations (e.g. `You are a senior...` vs `You are an expert...`), pick the best version and standardize.

3. **Replace** inline prompt strings in all 4 providers with `prompts.OracleSystemPrompt` (etc.).

4. **Delete** the duplicated string literals from each provider file.

**Files to touch:**
- Create: `gateway/prompts.go`
- Edit: `gateway/gemini.go`, `gateway/openai.go`, `gateway/anthropic.go`, `gateway/openrouter.go`

**Validation:** `go build ./...` — no test changes needed.

---

### 🔴 #2: Move `Standard` to `domain/`

**What:** `config.Standard` is a pure data type with no methods. It's imported by `gateway` (in `EvaluateCompliance`/`RefineSpecFile` signatures), creating a `gateway → config` layering violation.

**Steps:**

1. **Add** to `domain/domain.go`:
   ```go
   type Standard struct {
       ID           string   `yaml:"id" json:"id"`
       Name         string   `yaml:"name" json:"name"`
       Description  string   `yaml:"description" json:"description"`
       TargetFiles  []string `yaml:"target_files" json:"target_files"`
       Criteria     string   `yaml:"criteria" json:"criteria"`
       MinScore     int      `yaml:"min_score" json:"min_score"`
       ValidatorCmd string   `yaml:"validator_cmd,omitempty" json:"validator_cmd,omitempty"`
   }
   ```

2. **Add type alias** in `config/config.go` or `config/standards.go`:
   ```go
   type Standard = domain.Standard
   ```

3. **Update** `config.LoadStandards()` to return `[]domain.Standard`.

4. **Update** `gateway/gateway.go` — change `EvaluateCompliance` and `RefineSpecFile` signatures from `[]config.Standard` to `[]domain.Standard`.

5. **Update** all provider implementations and callers.

**Files to touch:** `domain/domain.go`, `config/config.go`, `gateway/*.go`, `generator/*.go`, `tui/dashboard/*.go`

**Validation:** `go build ./...` then `go test ./...`

---

### 🔴 #3: Remove `state → gateway` Import

**What:** `state/editor.go` imports `gateway` solely to use `gateway.Facts`, which is a type alias for `domain.Facts`.

**Steps:**

1. **Replace** `gateway.Facts` with `domain.Facts` in `state/editor.go`.

2. **Remove** the `github.com/toanle/synthspec/gateway` import.

**Files to touch:** `state/editor.go`

**Validation:** `go build ./state/...`

---

### 🔴 #4: Fix Silent Error Suppression in TUI

**What:** Multiple places swallow errors with `_`:
- `tui/dashboard/model.go` L147, L176: `standards, _ := config.LoadStandards()` and `templates, _ := config.LoadTemplates()`
- Multiple `cmd/` files: `settings, _ := config.LoadSettings()`
- `tui/dashboard/handlers.go` L79, L101, L195: `m.Session.Save()` unchecked

**Steps:**

1. In `tui/dashboard/model.go` — handle errors with logger:
   ```go
   standards, err := config.LoadStandards()
   if err != nil {
       logger.Log("WARN: failed to load standards: %v", err)
   }
   ```

2. In `cmd/*.go` — log silent suppression of `LoadSettings()` errors.

3. In `tui/dashboard/handlers.go` — check `Save()` return:
   ```go
   if err := m.Session.Save(); err != nil {
       logger.Log("session save failed: %v", err)
   }
   ```

**Files to touch:** `tui/dashboard/model.go`, `tui/dashboard/handlers.go`, `cmd/root.go`, `cmd/init.go`, `cmd/resume.go`, `cmd/update.go`

**Validation:** `go build ./...`

---

## Phase 2 — Structural Improvements (Medium effort, Medium-High impact)

### 🟠 #5: Generic `loadYAML[T]` Helper in `config`

**What:** `LoadStandards()`, `LoadTemplates()`, `LoadBlueprints()` are structurally identical — each reads embedded YAML, checks local override paths, and unmarshals.

**Steps:**

1. **Create** `config/yaml.go` with a generic helper:
   ```go
   func loadYAML[T any](embedded []byte, localPaths []string) (T, error)
   ```

2. **Refactor** all 3 loaders to use the helper.

**Files to touch:** Create `config/yaml.go`. Edit `config/config.go`.

**Validation:** `go test ./config/...`

---

### 🟠 #6: Split `config.go` into Focused Files

**What:** `config/config.go` is 264 lines handling 6 distinct concerns.

**Steps:**

1. **Create** `config/providers.go` — move `Provider*` constants, `Model*` constants, `autoDetectProvider()`, `getDefaultModel()`, `getAPIKeyForProvider()`, `LoadConfig()`, `Config` struct, `FilterApplicableStandards()`.

2. **Create** `config/standards.go` — move `Standard` (if not fully moved to domain), `StandardsConfig`, `LoadStandards()`, embedded YAML.

3. **Create** `config/templates.go` — move `Template`, `TemplatesConfig`, `LoadTemplates()`, embedded YAML.

4. **Create** `config/blueprints.go` — move `Blueprint`, `BlueprintFacts`, `BlueprintsConfig`, `LoadBlueprints()`, embedded YAML.

5. **Keep** `config/settings.go` as is (already focused).

6. **Shrink** `config/config.go` — it should only contain the `Config` struct and `LoadConfig()` if not moved, or be deleted entirely.

**Files to touch:** Create 4 new files. Edit/trim `config/config.go`.

**Validation:** `go build ./...` then `go test ./config/...`

---

### 🟠 #7: Move `getGatewayForSession` → `gateway/factory.go`

**What:** The provider switch-case in `cmd/init.go` (and duplicated in `cmd/root.go`) is a factory pattern living in command files.

**Steps:**

1. **Create** `gateway/factory.go` with:
   ```go
   func NewGateway(provider, apiKey, model string) (Gateway, error) { ... }
   ```

2. **Replace** switch-case in `cmd/init.go`, `cmd/root.go` (and any other place) with a call to `gateway.NewGateway(...)`.

**Files to touch:** Create `gateway/factory.go`. Edit `cmd/init.go`, `cmd/root.go`.

**Validation:** `go build ./...`

---

### 🟠 #8: Extract `resolveProjectName()` Helper in `cmd`

**What:** The ~20-line auto-detect block (list projects → check count → handle multi → return single) is duplicated in `resume.go`, `update.go`, `export.go`.

**Steps:**

1. **Create or add to** `cmd/helpers.go`:
   ```go
   func resolveProjectName(args []string, action string) (string, error) { ... }
   ```

2. **Replace** duplicated blocks in `resume.go`, `update.go`, `export.go`.

**Files to touch:** Create `cmd/helpers.go`. Edit `cmd/resume.go`, `cmd/update.go`, `cmd/export.go`.

**Validation:** `go build ./cmd/...`

---

## Phase 3 — Deeper Refactors (Medium-High effort)

### 🟡 #9: Add `Summarize()` to `Gateway` Interface

**What:** `state/pruner.go` abuses `QueryOracle`'s `NextQuestion` field to return conversation summaries. A dedicated method makes intent explicit.

**Steps:**

1. **Add** to `gateway/gateway.go`:
   ```go
   Summarize(ctx context.Context, history []Message) (string, error)
   ```

2. **Implement** in all 5 provider files (Gemini, OpenAI, Anthropic, OpenRouter, Mock).

3. **Update** `state/pruner.go` to call `gw.Summarize()` instead of `gw.QueryOracle()`.

4. **Remove** misuse of `NextQuestion` as summary channel.

**Files to touch:** `gateway/gateway.go`, all provider files, `state/pruner.go`

**Validation:** `go build ./...` then `go test ./state/...`

---

### 🟡 #10: Decouple Generator from `*state.Session`

**What:** The generator mutates `*state.Session` directly (calls `sess.Save()` internally), coupling synthesis to persistence.

**Steps:**

1. **Define** a `SessionPersistence` interface in `generator/` (or use a callback):
   ```go
   type SessionPersistence interface {
       SaveGeneratedFile(state generatedFileState) error
       LoadGeneratedFile(fileName string) (generatedFileState, bool)
   }
   ```

2. **Implement** the interface in `state/`.

3. **Inject** the interface into `fileGenerator` instead of `*state.Session`.

**Files to touch:** `generator/*.go`, `state/session.go`

**Validation:** `go build ./...` then `go test ./generator/...`

---

### 🟡 #11: Add TUI State Machine Tests

**What:** Dashboard and welcome have ~225 lines of mostly smoke tests. The interrogation/generation flow is untested.

**Steps:**

1. **Add tests** that call `Update()` with synthesized `tea.Msg` values — test state transitions, not rendering.

2. **Test key flows:**
   - Oracle result received → session state updated
   - Generation finished → status updated
   - Error handling paths

3. **Focus on model state assertions** (`m.loading`, `m.isGenerating`, `m.genStatus`, `m.isCompleted`).

**Files to touch:** `tui/dashboard/model_test.go`, `tui/dashboard/update_test.go`

**Validation:** `go test ./tui/dashboard/...`

---

### 🟡 #12: Add Hash Computation Unit Tests

**What:** `computeSha256` and `computeFactsHash` are pure functions — trivial to test but currently the test file is a 15-line placeholder.

**Steps:**

1. **Expand** `generator/session_state_test.go`:
   - `computeSha256`: test determinism, empty string, Unicode, large input
   - `computeFactsHash`: test with different facts, verify it changes when facts change

**Files to touch:** `generator/session_state_test.go`

**Validation:** `go test ./generator/... -run TestComputeSha256`

---

## Phase 4 — Low-Priority Polish

### 🔵 #13: Derive File List from Templates (not hardcoded)

**What:** `tui/dashboard/views_standards.go` has a hardcoded `synthesisFiles` slice that duplicates `config/templates.yaml`.

**Steps:**

1. **Remove** `synthesisFiles` variable.
2. **Change** the referencing functions to accept `[]config.Template` and derive file list from there.

**Files to touch:** `tui/dashboard/views_standards.go`, `tui/dashboard/views.go`

---

### 🔵 #14: Fix `ExportMetadata.Version` Hardcoding

**What:** `generator/export/exporter.go` L119 hardcodes `Version: "1.0.0"`.

**Steps:**

1. **Replace** `"1.0.0"` with `generator.EngineVersion`.

**Files to touch:** `generator/export/exporter.go`

---

### 🔵 #15: Remove Empty Directories

**What:** `generator/compliance/` and `generator/synthspec/` are empty.

**Steps:**

1. **Delete** `generator/compliance/` and `generator/synthspec/` directories.

---

### 🔵 #16: Fix Miscellaneous Naming Issues

| Location | Change |
|---|---|
| `generator/generator.go` | Rename `TelemetryMetadata` → `GenerationMetadata` |
| `state/session.go` | Replace `HasError bool + ErrorStr string` with `ErrMsg string` |
| `generator/generator.go` | Change `var EngineVersion = "dev"` → `const` ... actually needs to stay `var` for ldflags — leave as-is or add comment |
| `cmd/resume.go` L72 | Replace `"./output"` sentinel with named constant |
| `cmd/export.go` L102-108 | Replace manual bubble-sort with `sort.Slice` |
| `tui/welcome/model.go` | Unexport `IsNewProject` if possible |

---

### 🔵 #17: Evaluate `glamour` for Markdown Rendering

**What:** The hand-rolled markdown renderer (`tui/shared/markdown.go`, 275 lines) is untested and a maintenance burden.

**Steps:**

1. **Research** whether `glamour` is compatible with the current Bubbletea/Lipgloss setup.
2. **If compatible:** Replace `HighlightMarkdown()` calls with `glamour` rendering.
3. **If incompatible:** Add table-driven tests for each markdown feature as a short-term safety net.

---

## Execution Order

```
Phase 1 (Quick Wins)
  ├── #1: prompts.go extraction          ← START HERE (highest impact)
  ├── #2: Standard → domain/
  ├── #3: Remove state→gateway import
  └── #4: Fix silent error suppression

Phase 2 (Structural)
  ├── #5: Generic loadYAML[T] helper
  ├── #6: Split config.go
  ├── #7: gateway factory.go
  └── #8: resolveProjectName helper

Phase 3 (Deeper)
  ├── #9: Summarize() on Gateway
  ├── #10: Decouple generator from *state.Session
  ├── #11: TUI state machine tests
  └── #12: Hash computation tests

Phase 4 (Polish)
  ├── #13: Derive file list from templates
  ├── #14: Fix version hardcoding
  ├── #15: Remove empty dirs
  ├── #16: Naming fixes
  └── #17: Markdown renderer evaluation
```

Each phase should be completed and validated (`go build ./...`, `go test ./...`) before moving to the next.
