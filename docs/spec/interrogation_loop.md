# The Oracle Interrogation Loop

The Interactive Interrogation Loop (code-named "The Oracle") guides users from vague, high-level project summaries to a complete, formal specification structure.

## Core Mechanics

### 1. Single Question Constraint
The AI agent operates under a strict single-question constraint. It is prohibited from presenting multiple distinct questions or questions covering different topics in a single conversational turn. This helps keep the terminal UI clean and prevents cognitive overload for the user.

### 2. State & Confidence Scoring
The AI evaluates user inputs across four distinct specification vectors:
- **Functional**: Features, user stories, workflows, and edge cases.
- **Structural**: Component boundaries, data architecture, and protocol choices.
- **Security**: Key isolation, authorization models, and cryptography requirements.
- **Compliance**: Threat vectors, industry standards (e.g., SOC2, HIPAA), and regulatory constraints.

A localized state-scoring algorithm evaluates the conversation context and calculates a real-time completion confidence percentage (0% to 100%) for each dimension.

### 3. Verification Gate
The generation phase remains locked behind a compliance gate. The user cannot synthesize the final engineering assets until all four confidence meters hit **100%**.

## Special CLI Shell Commands

Mid-session, the user can invoke system-level commands in the input field:
- `:edit` - Forks a subprocess that opens the system default editor (e.g., VS Code, Vim, Notepad) to edit the raw compiled context JSON directly.
- `:exit` - Saves the current session state and exits the terminal.
- `:help` - Displays the command palette.
