# ADR-005: TUI Architecture with Bubbletea

## Status
Accepted

## Context
SynthSpec requires an interactive terminal UI for the interrogation loop (Oracle Q&A) and generation progress visualization. The UI must handle:
- Streaming token display from LLM
- Multi-step wizard flow (welcome → interrogation → generation → results)
- Keyboard and mouse input
- Real-time progress updates from background generation

## Decision
Use **Bubbletea** (Elm-inspired TUI framework) with the following structure:

### Package Organization
```
tui/
├── dashboard/          # Main interrogation/generation dashboard
│   ├── model.go        # Model struct + Init()
│   ├── update.go       # Update(msg) - pure state transitions
│   ├── commands.go     # Background commands (tea.Cmd)
│   ├── handlers.go     # Message handlers (Update delegates here)
│   ├── views*.go       # View rendering (View() delegates here)
│   └── keys/           # Key bindings
├── welcome/            # Startup screen
│   ├── model.go
│   ├── update.go
│   ├── views*.go
│   └── keys/
└── shared/             # Shared utilities
    ├── markdown.go     # Markdown rendering
    └── styles.go       # Lipgloss styles
```

### Bubbletea Patterns Used

#### Model-Update-View Loop
```go
type DashboardModel struct {
    Session   *state.Session
    Gateway   gateway.Gateway
    // ... embedded state structs
}

func (m DashboardModel) Init() tea.Cmd { ... }
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m DashboardModel) View() string { ... }
```

#### Embedded State Structs (Composition over Inheritance)
```go
type DashboardModel struct {
    GenerationState
    ThoughtStreamState
    ComplianceState
    ViewerState
    ApprovalGateState
    // ...
}
```

#### Background Commands (tea.Cmd)
```go
func (m DashboardModel) generateSpecsCmd(ctx context.Context) tea.Cmd {
    return func() tea.Msg {
        err := generator.Generate(ctx, m.Gateway, m.Session, m.OutputDir, m.genChan, m.approvalChan)
        return genFinishedMsg{err: err}
    }
}
```

#### Streaming Tokens
```go
func (m DashboardModel) queryOracleCmd(latestInput string) tea.Cmd {
    return func() tea.Msg {
        resp, err := m.Gateway.QueryOracleStream(ctx, m.Session.Facts, m.Session.History, latestInput, m.thoughtChan)
        return oracleResultMsg{resp: resp, err: err}
    }
}
```

### Key Bindings
- Defined in `keys/` subpackages
- Context-sensitive (different bindings in different modes)
- Help text generated dynamically

## Consequences

### Positive
- **Declarative**: State transitions are pure functions (easy to test)
- **Concurrency**: Background work via `tea.Cmd` integrates with event loop
- **Composability**: Embedded state structs keep `Update()` manageable
- **Ecosystem**: Bubbletea + Lipgloss + Bubbles provides rich TUI primitives
- **Cross-platform**: Works on Windows, macOS, Linux

### Negative
- **Learning curve**: Elm architecture unfamiliar to some Go developers
- **Verbosity**: Model/Update/View separation adds boilerplate
- **Testing**: Requires synthesizing `tea.Msg` for unit tests

### Neutral
- Mouse support optional but implemented for accessibility
- Markdown rendering uses custom parser (evaluated `glamour` - incompatible)

## Related
- ADR-001: Layered Architecture
- ADR-003: Source-First Generation Pipeline