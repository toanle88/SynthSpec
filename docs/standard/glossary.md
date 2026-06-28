# Domain Glossary

This glossary defines terms and vocabulary used throughout the SynthSpec ecosystem.

## Terminology

| Term | Definition |
| **BYOK (Bring Your Own Key)** | A security paradigm where users execute the software locally using their own API keys, preventing data from being routed through centralized servers. |
| **The Oracle** | The code-name for the interactive terminal interrogation loop that cross-examines users to complete requirement specifications. |
| **Source-First Synthesis** | The asset synthesis engine that generates `01_domain_model_use_cases.md` first, then fans out downstream documents in parallel from that locked source of truth. |
| **State Controller** | The backend component responsible for serializing transient conversation states to the local disk (`session.json`). |
| **LLM Provider Gateway** | The translation layer that formats core schemas into payloads compatible with Gemini, OpenAI, or Anthropic. |
| **STRIDE** | A threat modeling framework (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege) used to design SynthSpec's security controls. |
| **Context Pruning** | The automatic background process of summarizing Oracle chat logs when context length exceeds 75% of the target LLM limit. |
| **Fan-Out** | The downstream generation burst that runs `02-07` in parallel after the source domain document is locked. |
