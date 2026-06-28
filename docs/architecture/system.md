# System Architecture & Component Design

SynthSpec is architected as an event-driven, decoupled CLI system leveraging local state execution. It requires no central web backend, server infrastructure, or telemetry collection engines.

## Architecture Overview

```mermaid
graph TD
    User([Terminal User]) -->|CLI Commands| CLI[CLI Entrypoint & Router]
    CLI -->|Manages TUI Layout| TUI[TUI Dashboard Engine]
    TUI -->|Tracks Progress Metrics| State[State Controller]
    State -->|Reads/Writes| LocalDisk[(Local File System)]
    TUI -->|Dispatches Prompts| Gateway[LLM Provider Gateway]
    Gateway -->|BYOK Network Call| ExternalAPI{Upstream LLM Provider}
```

## Component Breakdown

### 1. CLI Entrypoint & Router
Responsible for processing root arguments, flags, and system commands. It reads execution contexts and overrides application runtime settings based on environment variables.

Key responsibilities:
- Parse flags and commands (`init`, `resume`, etc.).
- Bootstrap environment configuration (e.g., parsing API keys).
- Route control flow to either the TUI dashboard engine or the batch generator.

The batch generator now runs the domain source document first, then fans out the downstream documents in parallel using that locked source doc as the shared reference baseline.

### 2. State Controller
Maintains runtime synchronization. It tracks the dynamic conversation history matrix, evaluates completion scores across four dimensions (Functional, Structural, Security, Compliance), and serializes transient application states back to the local file system.

Key responsibilities:
- Maintain session state in memory.
- Serialize and deserialize state to/from a local `.synthspec/session.json` file.
- Track score calculation logic for progress metrics.

For details on the TUI Dashboard Engine, see [TUI Architecture](tui.md).
For details on the LLM Provider Gateway, see [LLM Provider Gateway Architecture](gateway.md).
