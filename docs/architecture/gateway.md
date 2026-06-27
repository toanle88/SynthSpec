# LLM Provider Gateway

The LLM Provider Gateway is an abstraction interface that converts unified application schemas into vendor-specific payload calls. This enables SynthSpec to switch dynamically between upstream LLMs without modifying core application logic.

## Multi-Model Routing

The system supports secure execution via locally defined environment variables for the following providers:
- **Gemini**: `GEMINI_API_KEY`
- **OpenAI**: `OPENAI_API_KEY`
- **Anthropic**: `ANTHROPIC_API_KEY`
- **OpenRouter**: `OPENROUTER_API_KEY`

Routing is determined dynamically via CLI flags (e.g., `--model gemini-1.5-pro` or `--model gpt-4o`). The Gateway abstracts away:
- System prompt payload formatting.
- Streaming structures.
- Provider-specific tool definitions and schema representations.
- Generation temperature overrides.

## Token Optimization & Context Pruning

To keep API costs low and prevent context overflow, the CLI tracks conversation history tokens. 

Before hitting **75% of the target LLM model’s context limit**, the gateway triggers a background summarization cycle. This:
1. Condenses answered requirements into a concise fact-based summary.
2. Flushes intermediate chat logs from the active LLM context.
3. Appends the condensed summary as the new conversation baseline.
4. Preserves engineering facts while freeing up context for continued interrogation.

## Rate Limiting & Backoff

To handle API rate throttling (HTTP 429), the gateway implements an automated **exponential backoff retry algorithm**. This ensures the TUI session remains robust even during periods of heavy API usage.
