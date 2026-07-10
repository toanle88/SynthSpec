package state

// modelLimits registers the context window sizes (in tokens) for supported models
var modelLimits = map[string]int{
	"gemini-2.5-pro":    2000000,
	"gemini-1.5-pro":    2000000,
	"gemini-1.5-flash":  1000000,
	"gpt-4o":            128000,
	"o3-mini":           200000,
	"claude-3-5-sonnet": 200000,
	"mock-model":        10000,
}

// GetModelLimit returns the context window size (in tokens) for a supported model
func GetModelLimit(model string) (int, bool) {
	limit, ok := modelLimits[model]
	return limit, ok
}
