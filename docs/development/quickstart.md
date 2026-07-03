# Quickstart Guide

Get up and running with SynthSpec in 5 minutes.

## Prerequisites

- **Go 1.26+** - [Download](https://go.dev/dl/)
- **Git** - [Download](https://git-scm.com/)
- **LLM API Key** - At least one of:
  - Google Gemini API key ([Get one](https://makersuite.google.com/app/apikey))
  - OpenAI API key ([Get one](https://platform.openai.com/api-keys))
  - Anthropic API key ([Get one](https://console.anthropic.com/))
  - OpenRouter API key ([Get one](https://openrouter.ai/keys))

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/toanle/synthspec.git
cd synthspec

# Build the binary
go build -o synthspec.exe main.go

# Verify it works
./synthspec.exe --help
```

### Using Go Install (Latest Release)

```bash
go install github.com/toanle/synthspec@latest
# Ensure ~/go/bin is in your PATH
synthspec --help
```

## Your First Project

### 1. Initialize a New Project

```bash
# Using Gemini (recommended for free tier)
./synthspec.exe init my-first-project --provider gemini --api-key $GEMINI_API_KEY

# Or using OpenAI
./synthspec.exe init my-first-project --provider openai --api-key $OPENAI_API_KEY

# Or using Anthropic
./synthspec.exe init my-first-project --provider anthropic --api-key $ANTHROPIC_API_KEY

# Or using OpenRouter (access to many models)
./synthspec.exe init my-first-project --provider openrouter --api-key $OPENROUTER_API_KEY
```

### 2. Interact with the Oracle

The TUI will launch. You'll see the **Oracle** asking questions to understand your project:

```
┌─ SynthSpec: my-first-project ────────────────────────────────────┐
│                                                                    │
│  🤖 Oracle: What are the primary user roles and functional        │
│             workflows of this application?                        │
│                                                                    │
│  1. Standard Admin, Editor, Viewer roles                          │
│  2. E-commerce Buyer and Seller workflows                         │
│  3. SaaS Tenant Owner and Member workflows                        │
│                                                                    │
│  > Type your answer or select a number...                         │
└────────────────────────────────────────────────────────────────────┘
```

**Tips:**
- Type free-form answers or select numbered choices
- Press `:edit` to open your `$EDITOR` for longer responses
- The Oracle builds understanding across 4 dimensions: Functional, Structural, Security, Compliance

### 3. Reach 100% Completion

Continue answering until all four dimensions reach 100%:

```
Functional: ████████████████████ 100%
Structural: ████████████████████ 100%
Security:   ████████████████████ 100%
Compliance: ████████████████████ 100%
```

### 4. Generate Specifications

Once complete, press **Enter** to start generation. The pipeline will:

1. **Generate source document** (domain model & use cases)
2. **Parallel fan-out** - generate 6 downstream documents simultaneously:
   - PRD (Functional Requirements)
   - System Architecture
   - API Architecture & Integration
   - Coding Standards & Guidelines
   - Security & Threat Model
   - Engineering Roadmap
3. **Self-correction loop** - each document validated for syntax & compliance
4. **Consistency verification** - cross-document logical consistency check
5. **Compliance report** - summary of all standards adherence

### 5. View Results

Generated files appear in `synthspec/my-first-project/output/`:

```
output/
├── 00_compliance_report.md      # Compliance summary
├── 01_domain_model_use_cases.md
├── 02_prd_functional.md
├── 03_system_architecture.md
├── 04_api_architecture_integration.md
├── 05_coding_standards_guidelines.md
├── 06_security_threat_model.md
├── 07_engineering_roadmap.md
└── .synthspec-meta.json         # Generation metadata
```

### 6. Export to HTML (Optional)

```bash
./synthspec.exe export my-first-project
# Creates synthspec/my-first-project/dist/index.html
```

Open `dist/index.html` in your browser for a searchable, navigable specification site with Mermaid diagrams rendered.

## Resuming Work

```bash
# Resume where you left off
./synthspec.exe resume my-first-project

# Or list all projects
./synthspec.exe list
```

## Using Blueprints

Jumpstart with pre-defined templates:

```bash
# List available blueprints
./synthspec.exe init --help  # Shows blueprint flag

# Initialize with a blueprint
./synthspec.exe init my-saas --provider gemini --api-key $KEY --blueprint saas-multi-tenant
```

## Configuration

### Global Settings

Settings are stored in `~/.config/synthspec/settings.yaml`:

```yaml
default_provider: gemini
default_model: gemini-2.5-pro
default_output_folder: ""
timeout_seconds: 300
max_retries: 3
debug: false
```

### Project-Level Overrides

Create `.synthspec.yaml` in your project directory to override settings per-project.

## Common Commands

| Command | Description |
|---------|-------------|
| `synthspec init <name>` | Create new project |
| `synthspec resume <name>` | Resume existing project |
| `synthspec update <name>` | Add/modify requirements |
| `synthspec export <name>` | Export to HTML |
| `synthspec list` | List all projects |
| `synthspec delete <name>` | Delete project |
| `synthspec --help` | Show all commands |

## Troubleshooting

### "API key not found"
Ensure your API key is set correctly:
```bash
export GEMINI_API_KEY="your-key-here"
./synthspec.exe init test --provider gemini --api-key $GEMINI_API_KEY
```

### "Context window exceeded"
The system automatically summarizes conversation history when token limits approach. If issues persist, start a fresh session with `update`.

### Generation fails
Check the compliance report (`00_compliance_report.md`) for details. Common issues:
- Missing API key permissions
- Rate limits (wait and retry)
- Model unavailable (try different model with `--model` flag)

### TUI rendering issues
Ensure your terminal supports:
- True color (24-bit)
- Unicode
- Mouse events (optional)

## Next Steps

- Read the [Architecture Overview](docs/architecture/system.md)
- Review [Development Guidelines](docs/development/contributing.md)
- Explore [Specification Documents](docs/spec/)
- Check [Standards & Glossary](docs/standard/)

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/toanle/synthspec/issues)
- **Discussions**: [GitHub Discussions](https://github.com/toanle/synthspec/discussions)
- **Documentation**: [docs/](docs/)

---

Happy specifying! 🚀