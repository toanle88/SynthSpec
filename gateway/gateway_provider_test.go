package gateway

import (
	"testing"
)

func TestNewAnthropicGateway(t *testing.T) {
	gw := NewAnthropicGateway("test-key", "")
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", gw.apiKey)
	}
	if gw.model != "claude-3-5-sonnet" {
		t.Errorf("expected default model 'claude-3-5-sonnet', got %q", gw.model)
	}
}

func TestNewAnthropicGateway_WithModel(t *testing.T) {
	gw := NewAnthropicGateway("test-key", "claude-3-opus")
	if gw.model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got %q", gw.model)
	}
}

func TestNewGeminiGateway(t *testing.T) {
	gw := NewGeminiGateway("test-key", "")
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", gw.apiKey)
	}
	if gw.model != "gemini-2.5-pro" {
		t.Errorf("expected default model 'gemini-2.5-pro', got %q", gw.model)
	}
}

func TestNewOpenAIGateway(t *testing.T) {
	gw := NewOpenAIGateway("test-key", "")
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", gw.apiKey)
	}
	if gw.model != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %q", gw.model)
	}
}

func TestNewOpenRouterGateway(t *testing.T) {
	gw := NewOpenRouterGateway("test-key", "")
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", gw.apiKey)
	}
	if gw.model != "meta-llama/llama-3.1-405b-instruct" {
		t.Errorf("expected default model 'meta-llama/llama-3.1-405b-instruct', got %q", gw.model)
	}
}
