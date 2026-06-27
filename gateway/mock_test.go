package gateway

import (
	"context"
	"testing"
)

func TestMockGatewayInterrogation(t *testing.T) {
	gw := NewMockGateway()
	ctx := context.Background()

	// Initial turn
	res, err := gw.QueryOracle(ctx, Facts{}, nil, "")
	if err != nil {
		t.Fatalf("failed to query oracle: %v", err)
	}

	if res.ConfidenceScores.Functional != 25 {
		t.Errorf("expected initial functional score to be 25, got %d", res.ConfidenceScores.Functional)
	}
	if res.NextQuestion == "" {
		t.Error("expected mock gateway to return a question")
	}

	// Complete turn (6 entries in history represents 3 full loops)
	history := []Message{
		{Role: "user", Content: "roles"},
		{Role: "assistant", Content: "question 1"},
		{Role: "user", Content: "storage"},
		{Role: "assistant", Content: "question 2"},
		{Role: "user", Content: "security"},
		{Role: "assistant", Content: "question 3"},
	}
	res2, err := gw.QueryOracle(ctx, Facts{}, history, "compliance")
	if err != nil {
		t.Fatalf("failed to query oracle: %v", err)
	}

	if res2.ConfidenceScores.Functional != 100 || res2.ConfidenceScores.Structural != 100 {
		t.Errorf("expected completed scores to be 100, got: %+v", res2.ConfidenceScores)
	}
	if res2.NextQuestion != "" {
		t.Errorf("expected next question to be empty on 100%% completion, got %q", res2.NextQuestion)
	}
}
