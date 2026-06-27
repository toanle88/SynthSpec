package config

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Supported Providers
const (
	ProviderGemini     = "gemini"
	ProviderOpenAI     = "openai"
	ProviderAnthropic  = "anthropic"
	ProviderOpenRouter = "openrouter"
	ProviderMock       = "mock"
)

// Default Models
const (
	DefaultModelGemini     = "gemini-2.5-pro"
	DefaultModelOpenAI     = "gpt-4o"
	DefaultModelAnthropic  = "claude-3-5-sonnet"
	DefaultModelOpenRouter = "meta-llama/llama-3.1-405b-instruct"
	DefaultModelMock       = "mock-model"
)

// Config holds the application configuration
type Config struct {
	Provider string
	Model    string
	APIKey   string
	Mock     bool
}

// LoadConfig resolves application configuration based on flags and env variables.
func LoadConfig(providerOverride, modelOverride string, mock bool) (*Config, error) {
	cfg := &Config{
		Mock: mock,
	}

	if mock {
		cfg.Provider = ProviderMock
		cfg.Model = DefaultModelMock
		return cfg, nil
	}

	// Auto-detect or use override
	geminiKey := os.Getenv("GEMINI_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")

	if providerOverride != "" {
		cfg.Provider = providerOverride
		switch providerOverride {
		case ProviderGemini:
			cfg.APIKey = geminiKey
		case ProviderOpenAI:
			cfg.APIKey = openaiKey
		case ProviderAnthropic:
			cfg.APIKey = anthropicKey
		case ProviderOpenRouter:
			cfg.APIKey = openrouterKey
		default:
			return nil, fmt.Errorf("unsupported provider: %s", providerOverride)
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key not set for specified provider %s", providerOverride)
		}
	} else {
		// Auto-detect precedence: Gemini > OpenAI > Anthropic > OpenRouter
		if geminiKey != "" {
			cfg.Provider = ProviderGemini
			cfg.APIKey = geminiKey
		} else if openaiKey != "" {
			cfg.Provider = ProviderOpenAI
			cfg.APIKey = openaiKey
		} else if anthropicKey != "" {
			cfg.Provider = ProviderAnthropic
			cfg.APIKey = anthropicKey
		} else if openrouterKey != "" {
			cfg.Provider = ProviderOpenRouter
			cfg.APIKey = openrouterKey
		} else {
			return nil, fmt.Errorf("no API keys found in environment. Please set GEMINI_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, or OPENROUTER_API_KEY")
		}
	}

	// Assign models (override or default)
	if modelOverride != "" {
		cfg.Model = modelOverride
	} else {
		switch cfg.Provider {
		case ProviderGemini:
			cfg.Model = DefaultModelGemini
		case ProviderOpenAI:
			cfg.Model = DefaultModelOpenAI
		case ProviderAnthropic:
			cfg.Model = DefaultModelAnthropic
		case ProviderOpenRouter:
			cfg.Model = DefaultModelOpenRouter
		}
	}

	return cfg, nil
}

//go:embed standards.yaml
var defaultStandardsYAML []byte

// Standard represents an engineering or quality standard
type Standard struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	TargetFiles []string `yaml:"target_files" json:"target_files"`
	Criteria    string   `yaml:"criteria" json:"criteria"`
	MinScore    int      `yaml:"min_score" json:"min_score"`
}

type StandardsConfig struct {
	Standards []Standard `yaml:"standards"`
}

// LoadStandards loads the standards from a local override file or falls back to the embedded defaults.
func LoadStandards() ([]Standard, error) {
	data := defaultStandardsYAML

	// Check for local overrides in order of preference
	overridePaths := []string{
		"standards.yaml",
		".synthspec/standards.yaml",
	}

	for _, p := range overridePaths {
		if _, err := os.Stat(p); err == nil {
			if fileData, readErr := os.ReadFile(p); readErr == nil {
				data = fileData
				break
			}
		}
	}

	var cfg StandardsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse standards configuration: %w", err)
	}

	return cfg.Standards, nil
}

