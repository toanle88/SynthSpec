package gateway

import (
	"fmt"

	"github.com/toanle/synthspec/config"
)

// NewGateway creates the appropriate Gateway implementation based on the provider name.
func NewGateway(provider, apiKey, model string) (Gateway, error) {
	switch provider {
	case config.ProviderMock:
		return NewMockGateway(), nil
	case config.ProviderGemini:
		return NewGeminiGateway(apiKey, model), nil
	case config.ProviderOpenAI:
		return NewOpenAIGateway(apiKey, model), nil
	case config.ProviderAnthropic:
		return NewAnthropicGateway(apiKey, model), nil
	case config.ProviderOpenRouter:
		return NewOpenRouterGateway(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unrecognized provider: %s", provider)
	}
}
