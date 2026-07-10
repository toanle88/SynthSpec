package config

import (
	"strings"
)

// ModelPricing defines the cost per 1M tokens in USD
type ModelPricing struct {
	PromptCostPer1M     float64
	CompletionCostPer1M float64
}

// pricingTable stores the default pricing for standard models.
var pricingTable = map[string]ModelPricing{
	// Gemini Models
	"gemini-2.5-pro": {PromptCostPer1M: 1.25, CompletionCostPer1M: 3.75},
	"gemini-2.5-flash": {PromptCostPer1M: 0.075, CompletionCostPer1M: 0.30},

	// OpenAI Models
	"gpt-4o":      {PromptCostPer1M: 2.50, CompletionCostPer1M: 10.00},
	"gpt-4o-mini": {PromptCostPer1M: 0.15, CompletionCostPer1M: 0.60},

	// Anthropic Models
	"claude-3-5-sonnet": {PromptCostPer1M: 3.00, CompletionCostPer1M: 15.00},
	"claude-3-opus":       {PromptCostPer1M: 15.00, CompletionCostPer1M: 75.00},
	"claude-3-5-haiku":   {PromptCostPer1M: 0.80, CompletionCostPer1M: 4.00},

	// OpenRouter Llama Models
	"meta-llama/llama-3.1-405b-instruct": {PromptCostPer1M: 2.66, CompletionCostPer1M: 2.66},
	"meta-llama/llama-3.1-70b-instruct":  {PromptCostPer1M: 0.52, CompletionCostPer1M: 0.75},

	// DeepSeek Models
	"deepseek/deepseek-v4-flash": {PromptCostPer1M: 0.09, CompletionCostPer1M: 0.18},

	// Mock Models
	"mock-model": {PromptCostPer1M: 0.0, CompletionCostPer1M: 0.0},
}

// CalculateCost estimates the USD cost of a query based on the model and token counts.
func CalculateCost(model string, promptTokens, completionTokens int) float64 {
	// Normalize model name (remove prefix or spaces if any)
	m := strings.TrimSpace(strings.ToLower(model))

	// Find exact match or standard defaults
	pricing, ok := pricingTable[m]
	if !ok {
		// Fallbacks based on prefix matches
		if strings.Contains(m, "gemini") {
			pricing = pricingTable["gemini-2.5-pro"]
		} else if strings.Contains(m, "gpt-4o-mini") {
			pricing = pricingTable["gpt-4o-mini"]
		} else if strings.Contains(m, "gpt-4") {
			pricing = pricingTable["gpt-4o"]
		} else if strings.Contains(m, "claude-3-5-sonnet") {
			pricing = pricingTable["claude-3-5-sonnet"]
		} else if strings.Contains(m, "claude-3-5-haiku") {
			pricing = pricingTable["claude-3-5-haiku"]
		} else if strings.Contains(m, "claude-3-opus") {
			pricing = pricingTable["claude-3-opus"]
		} else if strings.Contains(m, "llama-3.1-405b") {
			pricing = pricingTable["meta-llama/llama-3.1-405b-instruct"]
		} else if strings.Contains(m, "deepseek-v4-flash") {
			pricing = pricingTable["deepseek/deepseek-v4-flash"]
		} else {
			// Reasonable generic fallback (e.g. GPT-4o-like pricing)
			pricing = ModelPricing{PromptCostPer1M: 2.50, CompletionCostPer1M: 10.00}
		}
	}

	promptCost := (float64(promptTokens) / 1000000.0) * pricing.PromptCostPer1M
	completionCost := (float64(completionTokens) / 1000000.0) * pricing.CompletionCostPer1M

	return promptCost + completionCost
}
