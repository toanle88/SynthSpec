package gateway

import (
	"testing"

	"github.com/toanle/synthspec/config"
)

func TestNewGateway_Anthropic(t *testing.T) {
	gw, err := NewGateway(config.ProviderAnthropic, "test-key", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	baseGw, ok := gw.(*BaseGateway)
	if !ok {
		t.Fatalf("expected BaseGateway, got %T", gw)
	}
	adapter, ok := baseGw.adapter.(*AnthropicAdapter)
	if !ok {
		t.Fatalf("expected AnthropicAdapter, got %T", baseGw.adapter)
	}
	if adapter.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", adapter.apiKey)
	}
	if adapter.model != "claude-3-5-sonnet" {
		t.Errorf("expected default model 'claude-3-5-sonnet', got %q", adapter.model)
	}
}

func TestNewGateway_Gemini(t *testing.T) {
	gw, err := NewGateway(config.ProviderGemini, "test-key", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	baseGw, ok := gw.(*BaseGateway)
	if !ok {
		t.Fatalf("expected BaseGateway, got %T", gw)
	}
	adapter, ok := baseGw.adapter.(*GeminiAdapter)
	if !ok {
		t.Fatalf("expected GeminiAdapter, got %T", baseGw.adapter)
	}
	if adapter.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", adapter.apiKey)
	}
	if adapter.model != "gemini-2.5-pro" {
		t.Errorf("expected default model 'gemini-2.5-pro', got %q", adapter.model)
	}
}

func TestNewGateway_OpenAI(t *testing.T) {
	gw, err := NewGateway(config.ProviderOpenAI, "test-key", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	baseGw, ok := gw.(*BaseGateway)
	if !ok {
		t.Fatalf("expected BaseGateway, got %T", gw)
	}
	adapter, ok := baseGw.adapter.(*OpenAIAdapter)
	if !ok {
		t.Fatalf("expected OpenAIAdapter, got %T", baseGw.adapter)
	}
	if adapter.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", adapter.apiKey)
	}
	if adapter.model != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %q", adapter.model)
	}
}

func TestNewGateway_OpenRouter(t *testing.T) {
	gw, err := NewGateway(config.ProviderOpenRouter, "test-key", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	baseGw, ok := gw.(*BaseGateway)
	if !ok {
		t.Fatalf("expected BaseGateway, got %T", gw)
	}
	adapter, ok := baseGw.adapter.(*OpenRouterAdapter)
	if !ok {
		t.Fatalf("expected OpenRouterAdapter, got %T", baseGw.adapter)
	}
	if adapter.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", adapter.apiKey)
	}
	if adapter.model != "meta-llama/llama-3.1-405b-instruct" {
		t.Errorf("expected default model 'meta-llama/llama-3.1-405b-instruct', got %q", adapter.model)
	}
}

func TestBuildExtractStructuralEntitiesRequest_AllAdapters(t *testing.T) {
	dummyDoc := "# Dummy Domain Model\n- Workflow: Test"

	// 1. Gemini
	gwGemini, _ := NewGateway(config.ProviderGemini, "test-key", "")
	baseGemini := gwGemini.(*BaseGateway)
	reqGemini, err := baseGemini.adapter.BuildExtractStructuralEntitiesRequest(dummyDoc)
	if err != nil {
		t.Errorf("Gemini BuildExtractStructuralEntitiesRequest error: %v", err)
	}
	if reqGemini == nil {
		t.Error("Gemini BuildExtractStructuralEntitiesRequest returned nil request")
	}

	// 2. OpenAI
	gwOpenAI, _ := NewGateway(config.ProviderOpenAI, "test-key", "")
	baseOpenAI := gwOpenAI.(*BaseGateway)
	reqOpenAI, err := baseOpenAI.adapter.BuildExtractStructuralEntitiesRequest(dummyDoc)
	if err != nil {
		t.Errorf("OpenAI BuildExtractStructuralEntitiesRequest error: %v", err)
	}
	if reqOpenAI == nil {
		t.Error("OpenAI BuildExtractStructuralEntitiesRequest returned nil request")
	}

	// 3. Anthropic
	gwAnthropic, _ := NewGateway(config.ProviderAnthropic, "test-key", "")
	baseAnthropic := gwAnthropic.(*BaseGateway)
	reqAnthropic, err := baseAnthropic.adapter.BuildExtractStructuralEntitiesRequest(dummyDoc)
	if err != nil {
		t.Errorf("Anthropic BuildExtractStructuralEntitiesRequest error: %v", err)
	}
	if reqAnthropic == nil {
		t.Error("Anthropic BuildExtractStructuralEntitiesRequest returned nil request")
	}

	// 4. OpenRouter
	gwOpenRouter, _ := NewGateway(config.ProviderOpenRouter, "test-key", "")
	baseOpenRouter := gwOpenRouter.(*BaseGateway)
	reqOpenRouter, err := baseOpenRouter.adapter.BuildExtractStructuralEntitiesRequest(dummyDoc)
	if err != nil {
		t.Errorf("OpenRouter BuildExtractStructuralEntitiesRequest error: %v", err)
	}
	if reqOpenRouter == nil {
		t.Error("OpenRouter BuildExtractStructuralEntitiesRequest returned nil request")
	}
}

func TestBuildOptimizePromptRequest_AllAdapters(t *testing.T) {
	dummyFiles := map[string]string{
		"01_domain_model.md": "# Domain Model",
	}

	// 1. Gemini
	gwGemini, _ := NewGateway(config.ProviderGemini, "test-key", "")
	baseGemini := gwGemini.(*BaseGateway)
	reqGemini, err := baseGemini.adapter.BuildOptimizePromptRequest(dummyFiles)
	if err != nil {
		t.Errorf("Gemini BuildOptimizePromptRequest error: %v", err)
	}
	if reqGemini == nil {
		t.Error("Gemini BuildOptimizePromptRequest returned nil request")
	}

	// 2. OpenAI
	gwOpenAI, _ := NewGateway(config.ProviderOpenAI, "test-key", "")
	baseOpenAI := gwOpenAI.(*BaseGateway)
	reqOpenAI, err := baseOpenAI.adapter.BuildOptimizePromptRequest(dummyFiles)
	if err != nil {
		t.Errorf("OpenAI BuildOptimizePromptRequest error: %v", err)
	}
	if reqOpenAI == nil {
		t.Error("OpenAI BuildOptimizePromptRequest returned nil request")
	}

	// 3. Anthropic
	gwAnthropic, _ := NewGateway(config.ProviderAnthropic, "test-key", "")
	baseAnthropic := gwAnthropic.(*BaseGateway)
	reqAnthropic, err := baseAnthropic.adapter.BuildOptimizePromptRequest(dummyFiles)
	if err != nil {
		t.Errorf("Anthropic BuildOptimizePromptRequest error: %v", err)
	}
	if reqAnthropic == nil {
		t.Error("Anthropic BuildOptimizePromptRequest returned nil request")
	}

	// 4. OpenRouter
	gwOpenRouter, _ := NewGateway(config.ProviderOpenRouter, "test-key", "")
	baseOpenRouter := gwOpenRouter.(*BaseGateway)
	reqOpenRouter, err := baseOpenRouter.adapter.BuildOptimizePromptRequest(dummyFiles)
	if err != nil {
		t.Errorf("OpenRouter BuildOptimizePromptRequest error: %v", err)
	}
	if reqOpenRouter == nil {
		t.Error("OpenRouter BuildOptimizePromptRequest returned nil request")
	}
}

