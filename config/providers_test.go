package config

import (
	"testing"
)

func TestLoadConfig_Mock(t *testing.T) {
	cfg, err := LoadConfig("", "", true)
	if err != nil {
		t.Fatalf("LoadConfig with mock=true should succeed: %v", err)
	}
	if cfg.Provider != ProviderMock {
		t.Errorf("expected provider %q, got %q", ProviderMock, cfg.Provider)
	}
	if cfg.Model != DefaultModelMock {
		t.Errorf("expected model %q, got %q", DefaultModelMock, cfg.Model)
	}
	if !cfg.Mock {
		t.Errorf("expected Mock=true")
	}
}

func TestLoadConfig_OverrideWithKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key-gemini")
	cfg, err := LoadConfig(ProviderGemini, "", false)
	if err != nil {
		t.Fatalf("LoadConfig with override+key should succeed: %v", err)
	}
	if cfg.Provider != ProviderGemini {
		t.Errorf("expected provider %q, got %q", ProviderGemini, cfg.Provider)
	}
	if cfg.APIKey != "test-key-gemini" {
		t.Errorf("expected API key %q, got %q", "test-key-gemini", cfg.APIKey)
	}
	if cfg.Model != DefaultModelGemini {
		t.Errorf("expected default model %q, got %q", DefaultModelGemini, cfg.Model)
	}
}

func TestLoadConfig_OverrideNoKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	_, err := LoadConfig(ProviderGemini, "", false)
	if err == nil {
		t.Fatal("LoadConfig with override but no key should fail")
	}
}

func TestLoadConfig_AutoDetectOrder(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "key-openai")
	t.Setenv("GEMINI_API_KEY", "key-gemini")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect should succeed: %v", err)
	}
	if cfg.Provider != ProviderGemini {
		t.Errorf("expected Gemini to have priority, got %q", cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "key-openai")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect OpenAI should succeed: %v", err)
	}
	if cfg.Provider != ProviderOpenAI {
		t.Errorf("expected provider %q, got %q", ProviderOpenAI, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "key-anthropic")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect Anthropic should succeed: %v", err)
	}
	if cfg.Provider != ProviderAnthropic {
		t.Errorf("expected provider %q, got %q", ProviderAnthropic, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectOpenRouter(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "key-openrouter")
	cfg, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("LoadConfig auto-detect OpenRouter should succeed: %v", err)
	}
	if cfg.Provider != ProviderOpenRouter {
		t.Errorf("expected provider %q, got %q", ProviderOpenRouter, cfg.Provider)
	}
}

func TestLoadConfig_AutoDetectNoKeys(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	_, err := LoadConfig("", "", false)
	if err == nil {
		t.Fatal("LoadConfig with no keys should fail")
	}
}

func TestLoadConfig_ModelOverride(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	cfg, err := LoadConfig(ProviderGemini, "custom-model", false)
	if err != nil {
		t.Fatalf("LoadConfig with model override should succeed: %v", err)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("expected model %q, got %q", "custom-model", cfg.Model)
	}
}

func TestAutoDetectProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gk")
	t.Setenv("OPENAI_API_KEY", "ok")
	prov, key := autoDetectProvider()
	if prov != ProviderGemini {
		t.Errorf("expected Gemini priority, got %q", prov)
	}
	if key != "gk" {
		t.Errorf("expected key 'gk', got %q", key)
	}
}

func TestAutoDetectProvider_None(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	prov, key := autoDetectProvider()
	if prov != "" || key != "" {
		t.Errorf("expected empty provider and key, got %q / %q", prov, key)
	}
}

func TestGetAPIKeyForProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gk")
	t.Setenv("OPENAI_API_KEY", "ok")
	t.Setenv("ANTHROPIC_API_KEY", "ak")
	t.Setenv("OPENROUTER_API_KEY", "rk")

	tests := []struct {
		provider string
		want     string
	}{
		{ProviderGemini, "gk"},
		{ProviderOpenAI, "ok"},
		{ProviderAnthropic, "ak"},
		{ProviderOpenRouter, "rk"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := getAPIKeyForProvider(tt.provider)
		if got != tt.want {
			t.Errorf("getAPIKeyForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestGetDefaultModel(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{ProviderGemini, DefaultModelGemini},
		{ProviderOpenAI, DefaultModelOpenAI},
		{ProviderAnthropic, DefaultModelAnthropic},
		{ProviderOpenRouter, DefaultModelOpenRouter},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := getDefaultModel(tt.provider)
		if got != tt.want {
			t.Errorf("getDefaultModel(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}
