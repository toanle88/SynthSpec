# Performance Rules & Resource Limits

Since SynthSpec executes locally as a CLI tool, performance guidelines focus on TUI responsiveness, file I/O safety, and API payload efficiency.

## 1. Startup Latency
- The CLI should boot and render the initial TUI dashboard in **under 200ms**.
- External configuration files and environment keys must be loaded lazily or immediately parsed on boot without blocking TUI initiation.

## 2. Asynchronous TUI Rendering
- Screen refreshes must not block on LLM gateway requests.
- The UI main loop must listen to user inputs concurrently with active background processes (e.g. running a progress spinner on a separate thread).

## 3. Summarization Threshold (Context Pruning)
- To prevent slow LLM completions or context overflow:
  - Track conversation history token count.
  - Trigger a background summarization cycle once history exceeds **75%** of the model's token limit.
- Keep the condensed summary under **1000 tokens** to leave maximum workspace capacity for subsequent interrogation prompts.
