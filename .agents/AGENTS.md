# SynthSpec Agent Rules
- This is a Go CLI project using Bubbletea for TUI
- Follow layered architecture: domain -> config -> gateway/state -> generator -> cmd/tui
- Keep files under 350 lines
- All packages must have test files
- Use interfaces for cross-package dependencies
