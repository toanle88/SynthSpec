# General Coding Standards

This document describes the general coding policies and guidelines for all SynthSpec contributors.

## General Design Policies

1. **Keep it Local**: SynthSpec is a local, privacy-first tool. Avoid introducing network operations outside the LLM Gateway layer.
2. **Event-Driven TUI**: Keep TUI rendering code decoupled from state manipulation. TUI updates should be triggered by State changes or user interaction events.
3. **Decoupled Gateway**: Any additions to LLM logic should be implemented via the standard gateway interfaces. Do not reference vendor-specific SDKs outside the `gateway/` package.
4. **Fresh Prompt Retries**: Retry loops should rebuild prompts from the current artifact and source-of-truth context instead of replaying prior retry transcripts.

## Version Control and Commit Standards

- Write clear, imperative Git commit messages (e.g., `feat: add Anthropic gateway integration`, `fix: correct state metric calculations`).
- Prefix commits using standard tags: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`.
