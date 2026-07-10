package gateway

import (
	"fmt"
	"time"

	"github.com/toanle/synthspec/config"
)

// NewGateway creates the appropriate Gateway implementation based on the provider name.
func NewGateway(provider, apiKey, model string) (Gateway, error) {
	var adapter ProviderAdapter
	timeout := 5 * time.Minute
	maxRetries := 3

	if s, err := config.LoadSettings(); err == nil && s != nil {
		timeout = time.Duration(s.TimeoutSeconds) * time.Second
		maxRetries = s.MaxRetries
	}

	switch provider {
	case config.ProviderMock:
		return NewMockGateway(), nil // Mock implements Gateway directly
	case config.ProviderGemini:
		adapter = NewGeminiAdapter(apiKey, model)
	case config.ProviderOpenAI:
		adapter = NewOpenAIAdapter(apiKey, model)
	case config.ProviderAnthropic:
		adapter = NewAnthropicAdapter(apiKey, model)
	case config.ProviderOpenRouter:
		adapter = NewOpenRouterAdapter(apiKey, model)
	default:
		return nil, fmt.Errorf("unrecognized provider: %s", provider)
	}

	return NewBaseGateway(adapter, timeout, maxRetries), nil
}
