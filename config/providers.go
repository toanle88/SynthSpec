package config

import (
	"fmt"
	"os"
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

func getAPIKeyForProvider(provider string) string {
	switch provider {
	case ProviderGemini:
		return os.Getenv("GEMINI_API_KEY")
	case ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case ProviderOpenRouter:
		return os.Getenv("OPENROUTER_API_KEY")
	}
	return ""
}

func autoDetectProvider() (string, string) {
	if k := os.Getenv("GEMINI_API_KEY"); k != "" {
		return ProviderGemini, k
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		return ProviderOpenAI, k
	}
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		return ProviderAnthropic, k
	}
	if k := os.Getenv("OPENROUTER_API_KEY"); k != "" {
		return ProviderOpenRouter, k
	}
	return "", ""
}

func getDefaultModel(provider string) string {
	switch provider {
	case ProviderGemini:
		return DefaultModelGemini
	case ProviderOpenAI:
		return DefaultModelOpenAI
	case ProviderAnthropic:
		return DefaultModelAnthropic
	case ProviderOpenRouter:
		return DefaultModelOpenRouter
	}
	return ""
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

	if providerOverride != "" {
		cfg.Provider = providerOverride
		cfg.APIKey = getAPIKeyForProvider(providerOverride)
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key not set for specified provider %s", providerOverride)
		}
	} else {
		p, k := autoDetectProvider()
		if p == "" {
			return nil, fmt.Errorf("no API keys found in environment. Please set GEMINI_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, or OPENROUTER_API_KEY")
		}
		cfg.Provider = p
		cfg.APIKey = k
	}

	if modelOverride != "" {
		cfg.Model = modelOverride
	} else {
		cfg.Model = getDefaultModel(cfg.Provider)
	}

	return cfg, nil
}
