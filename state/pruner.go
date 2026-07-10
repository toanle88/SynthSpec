package state

import (
	"context"
	"fmt"

	"github.com/toanle/synthspec/domain"
)

// ContextSummarizer provides an interface to summarize conversation history.
type ContextSummarizer interface {
	Summarize(ctx context.Context, history []domain.Message) (string, error)
}

// CheckAndPruneContext evaluates total tokens and runs summarization if over 75% capacity
func (s *Session) CheckAndPruneContext(ctx context.Context, gw ContextSummarizer) (bool, error) {
	limit, exists := GetModelLimit(s.Model)
	if !exists {
		// Default conservative limit
		limit = 100000
	}

	threshold := int(float64(limit) * 0.75)
	if s.TotalTokensUsed <= threshold {
		return false, nil
	}

	// Summarize conversation history using dedicated Summarize method
	summaryText, err := gw.Summarize(ctx, s.History)
	if err != nil {
		return false, fmt.Errorf("summarization call failed: %w", err)
	}

	// Reset conversation history to a single condensed context block
	if summaryText == "" {
		summaryText = "Summarized historical progress."
	}

	s.History = []domain.Message{
		{Role: "user", Content: "Let's summarize our progress so far."},
		{Role: "assistant", Content: "Summary of earlier conversation:\n" + summaryText},
	}
	// Note: We don't add tokens for summarization since it's a separate call

	if err := s.Save(); err != nil {
		return true, fmt.Errorf("failed to save session after pruning: %w", err)
	}

	return true, nil
}
