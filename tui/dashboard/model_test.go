package dashboard

import (
	"testing"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

func TestNewDashboardModel(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw := gateway.NewMockGateway()

	m := NewDashboardModel(sess, gw, "")
	if m.Session == nil {
		t.Error("expected non-nil Session")
	}
	if m.Gateway == nil {
		t.Error("expected non-nil Gateway")
	}
	if m.isCompleted {
		t.Errorf("expected isCompleted=false for new session")
	}
}

func TestDashboardModel_InitReturnsCmd(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash-init",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw := gateway.NewMockGateway()
	m := NewDashboardModel(sess, gw, "")

	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestDashboardModel_Completion(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash-complete",
		Provider:    "mock",
		Model:       "mock-model",
		Facts: gateway.Facts{
			Functional: "test",
			Structural: "test",
		},
		Scores: gateway.ConfidenceScores{
			Functional: 100,
			Structural: 100,
			Security:   100,
			Compliance: 100,
		},
	}
	gw := gateway.NewMockGateway()
	m := NewDashboardModel(sess, gw, "")
	if !m.isCompleted {
		t.Error("expected isCompleted=true when all scores are 100")
	}
}

func TestCheckCompletion(t *testing.T) {
	tests := []struct {
		name   string
		scores gateway.ConfidenceScores
		want   bool
	}{
		{"all 100", gateway.ConfidenceScores{Functional: 100, Structural: 100, Security: 100, Compliance: 100}, true},
		{"all zero", gateway.ConfidenceScores{Functional: 0, Structural: 0, Security: 0, Compliance: 0}, false},
		{"partial", gateway.ConfidenceScores{Functional: 100, Structural: 80, Security: 100, Compliance: 100}, false},
		{"one not 100", gateway.ConfidenceScores{Functional: 100, Structural: 100, Security: 100, Compliance: 99}, false},
		{"all above 100", gateway.ConfidenceScores{Functional: 150, Structural: 120, Security: 110, Compliance: 105}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCompletion(tt.scores)
			if got != tt.want {
				t.Errorf("checkCompletion(%+v) = %v, want %v", tt.scores, got, tt.want)
			}
		})
	}
}
