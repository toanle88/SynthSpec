# Maintainer Runbooks & Troubleshooting

This runbook outlines operational procedures for development, troubleshooting local session states, and handling upstream API outages.

## Development Setup

1. **Prerequisites**: Go 1.20+ installed.
2. **Cloning & Build**:
   ```bash
   git clone https://github.com/your-org/synthspec.git
   cd synthspec
   go build -o synthspec main.go
   ```
3. **Environment Setup**: Define at least one API key:
   ```bash
   # Windows (PowerShell)
   $env:GEMINI_API_KEY="your-key-here"
   $env:OPENROUTER_API_KEY="your-key-here"

   # macOS / Linux
   export GEMINI_API_KEY="your-key-here"
   export OPENROUTER_API_KEY="your-key-here"
   ```

## Troubleshooting Local Session Corruptions

If a user reports that a session is stuck or failing to resume via `synthspec resume`:
1. Check the local `.synthspec/` state folder in their project.
2. Locate the `.synthspec/session.json` file.
3. Validate the file against standard JSON syntax.
4. If corrupt:
   - Make a backup of the file.
   - Run the state formatter tool `synthspec repair` (if implemented) or manually fix missing closing brackets or fields in their default text editor.

## Upstream API Failures

If upstream LLM endpoints throw HTTP 500s or 503s:
- Validate whether user key billing limits are exceeded.
- Recommend switching models using `--model` flags or switching API providers if a backup key exists (e.g. switching from Anthropic to Gemini).
