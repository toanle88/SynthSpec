package state

import (
	"context"
	"fmt"

	"github.com/toanle/synthspec/gateway"
)

// CheckAndPruneContext evaluates total tokens and runs summarization if over 75% capacity
func (s *Session) CheckAndPruneContext(ctx context.Context, gw gateway.Gateway) (bool, error) {
	limit, exists := ModelLimits[s.Model]
	if !exists {
		// Default conservative limit
		limit = 100000
	}

	threshold := int(float64(limit) * 0.75)
	if s.TotalTokensUsed <= threshold {
		return false, nil
	}

	// Summarize conversation history
	summaryPrompt := "Summarize the key architectural choices, user preferences, and engineering requirements established in this chat history. Compress it into a clear, single paragraph summarizing the consensus."

	// Create a temporary history for summarization
	sumHistory := append(s.History, gateway.Message{Role: "user", Content: summaryPrompt})

	resp, err := gw.QueryOracle(ctx, s.Facts, sumHistory, "")
	if err != nil {
		return false, fmt.Errorf("summarization call failed: %w", err)
	}

	// Reset conversation history to a single condensed context block
	summaryText := "Summary of earlier conversation:\n" + resp.NextQuestion // Using next_question as the return channel in standard QueryOracle
	if summaryText == "" {
		summaryText = "Summarized historical progress."
	}

	s.History = []gateway.Message{
		{Role: "user", Content: "Let's summarize our progress so far."},
		{Role: "assistant", Content: summaryText},
	}
	s.TotalTokensUsed += resp.TokensPrompt + resp.TokensCompletion

	if err := s.Save(); err != nil {
		return true, fmt.Errorf("failed to save session after pruning: %w", err)
	}

	return true, nil
}
