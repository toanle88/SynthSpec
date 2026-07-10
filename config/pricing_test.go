package config

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		expectedCost     float64
	}{
		{
			name:             "OpenAI GPT-4o exact match",
			model:            "gpt-4o",
			promptTokens:     1000000,
			completionTokens: 1000000,
			expectedCost:     12.50, // 2.50 + 10.00
		},
		{
			name:             "Gemini 2.5 Pro exact match",
			model:            "gemini-2.5-pro",
			promptTokens:     1000000,
			completionTokens: 1000000,
			expectedCost:     5.00, // 1.25 + 3.75
		},
		{
			name:             "DeepSeek V4 Flash exact match",
			model:            "deepseek/deepseek-v4-flash",
			promptTokens:     1000000,
			completionTokens: 1000000,
			expectedCost:     0.27, // 0.09 + 0.18
		},
		{
			name:             "DeepSeek V4 Flash fallback",
			model:            "deepseek-v4-flash",
			promptTokens:     500000,
			completionTokens: 500000,
			expectedCost:     0.135, // 0.045 + 0.09
		},
		{
			name:             "OpenRouter Llama fallback",
			model:            "llama-3.1-405b",
			promptTokens:     1000000,
			completionTokens: 1000000,
			expectedCost:     5.32, // 2.66 + 2.66
		},
		{
			name:             "Unrecognized fallback to GPT-4o-like",
			model:            "some-random-model",
			promptTokens:     1000000,
			completionTokens: 1000000,
			expectedCost:     12.50, // 2.50 + 10.00
		},
		{
			name:             "Mock model is free",
			model:            "mock-model",
			promptTokens:     9999999,
			completionTokens: 9999999,
			expectedCost:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCost(tt.model, tt.promptTokens, tt.completionTokens)
			// Using small epsilon for floating-point comparison
			if math.Abs(got-tt.expectedCost) > 1e-9 {
				t.Errorf("CalculateCost(%q, %d, %d) = %f; want %f", tt.model, tt.promptTokens, tt.completionTokens, got, tt.expectedCost)
			}
		})
	}
}
