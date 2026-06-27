# Error Handling Policies

SynthSpec requires robust error handling to prevent TUI crashes, local file corruption, or API credential leaks.

## Error Handling Standards

### 1. API Call Failures
- Wrap API transport errors cleanly.
- Never output raw API key values in log files or standard error dumps.
- For rate limits (HTTP 429), standard gateway code must execute exponential backoff retries rather than crashing.

### 2. File I/O & Session Failures
- If state saving to `.synthspec/session.json` fails, warn the user inside the TUI viewport and retry writing to a backup path (e.g., `session.json.bak`).
- Never panic on deserialization errors; fallback to a safe state restore and report validation failures cleanly.

### 3. TUI Crash Recovery
- Run the TUI loop within a recovery block (`recover()` in Go) to ensure that if a panic occurs, the terminal screen configuration is restored to standard scrolling mode before printing the stack trace. This prevents terminal corruption for the user.
