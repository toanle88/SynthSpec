# TUI Dashboard Engine

The TUI (Terminal User Interface) Dashboard Engine is built using an asynchronous terminal framework in Go. It controls terminal display state, isolates standard output streams from system errors, renders layout regions, and intercepts keyboard event loops.

## Layout and Regions

The TUI provides visual feedback indicators instead of standard linear scrolling logs. It renders distinct interface zones:

1. **Header Panel**: Displays project metadata, session elapsed time, and system state.
2. **Categorical Completion Meters**: Shows live status bar updates for internal specification dimensions (Functional, Structural, Security, Compliance).
3. **Current Question Panel**: Displays the single question being asked by the AI Oracle.
4. **User Prompt Input Field**: Accepts interactive input from the user. Includes command processing for inline actions.
5. **Asset Generation Pane**: When synthesis begins, the UI separates the locked source document from the parallel downstream fan-out and shows a phase label for each stage.

## User Interactions & Keybindings

- **Normal Input**: Typing directly into the input field to answer questions.
- **`:edit` command**: Forks a subprocess opening the system default text editor (`$EDITOR` or `notepad`) to modify the compiled context JSON.
- **`ctrl+c` / `:exit`**: Safely saves the session state to disk via the State Controller and exits the CLI.

## Latency Indicators

Because deep architectural reasoning requires extended LLM execution times, the TUI must display a dynamic, non-blocking loading spinner to confirm system responsiveness during API round-trips. This spinner runs on a separate rendering thread to prevent blocking user key inputs or freezing the UI.

During synthesis, the progress view groups `01_domain_model_use_cases.md` as the source section and renders the remaining documents as a parallel batch so the fan-out is easy to scan.
