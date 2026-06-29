package state

// ModelLimits registers the context window sizes (in tokens) for supported models
var ModelLimits = map[string]int{
	"gemini-2.5-pro":    2000000,
	"gemini-1.5-pro":    2000000,
	"gemini-1.5-flash":  1000000,
	"gpt-4o":            128000,
	"o3-mini":           200000,
	"claude-3-5-sonnet": 200000,
	"mock-model":        10000,
}
