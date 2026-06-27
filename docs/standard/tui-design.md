# TUI Design Standards

Since SynthSpec operates entirely in the terminal, visual styling, spacing, and layout are constrained by character-cell terminal grids. The following guidelines define TUI layouts to ensure readability, responsiveness, and premium aesthetic appeal.

## Color Palette

Ensure colors are readable on both dark and light terminal backgrounds. Prefer terminal-agnostic ANSI colors or high-contrast custom hex codes if using standard true-color terms:
- **Primary / Prompts**: Teal or Cyan (`#00E5FF` / ANSI 14) for actions and active questions.
- **Success / 100% Meters**: Emerald Green (`#00E676` / ANSI 10) for completed categories and success confirmations.
- **Warnings / Loading**: Amber or Yellow (`#FFD700` / ANSI 11) for background operations, rate-limiting notifications, and loading spinners.
- **Muted Elements**: Gray (`#757575` / ANSI 8) for borders, secondary descriptions, and timestamps.

## Layout Hierarchy

The UI should fit comfortably within an **80x24 character grid** (the standard baseline CLI resolution).
- **Header**: Single-line title and session status.
- **Body**: Split-screen panel layout.
  - Left panel: Live Categorical Completion Meters (Progress bars).
  - Right panel: Interrogation logs and current active question.
- **Footer**: Input field preceded by a distinct prompt character (e.g., `synthspec > `).

## Dynamic Indicators

- **Loading Spinner**: A non-blocking character animation (e.g., `| / - \`) shown while waiting for LLM Provider responses.
- **Progress Bars**: Represent percentage completion visually using filled block characters (e.g., `█` or `░`).
