package config

import (
	"fmt"
	"os"
)

// Supported Providers
const (
	ProviderGemini    = "gemini"
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderMock      = "mock"
)

// Default Models
const (
	DefaultModelGemini    = "gemini-2.5-pro"
	DefaultModelOpenAI    = "gpt-4o"
	DefaultModelAnthropic = "claude-3-5-sonnet"
	DefaultModelMock      = "mock-model"
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

	if providerOverride != "" {
		cfg.Provider = providerOverride
		switch providerOverride {
		case ProviderGemini:
			cfg.APIKey = geminiKey
		case ProviderOpenAI:
			cfg.APIKey = openaiKey
		case ProviderAnthropic:
			cfg.APIKey = anthropicKey
		default:
			return nil, fmt.Errorf("unsupported provider: %s", providerOverride)
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key not set for specified provider %s", providerOverride)
		}
	} else {
		// Auto-detect precedence: Gemini > OpenAI > Anthropic
		if geminiKey != "" {
			cfg.Provider = ProviderGemini
			cfg.APIKey = geminiKey
		} else if openaiKey != "" {
			cfg.Provider = ProviderOpenAI
			cfg.APIKey = openaiKey
		} else if anthropicKey != "" {
			cfg.Provider = ProviderAnthropic
			cfg.APIKey = anthropicKey
		} else {
			return nil, fmt.Errorf("no API keys found in environment. Please set GEMINI_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY")
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
		}
	}

	return cfg, nil
}
