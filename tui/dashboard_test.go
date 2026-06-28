package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestDashboardModel_ViewReturnsContent(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash-view",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw := gateway.NewMockGateway()
	m := NewDashboardModel(sess, gw, "")

	view := m.View()
	if view == "" {
		t.Error("expected non-empty View")
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

func TestDashboardModel_WindowSize(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash-ws",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw := gateway.NewMockGateway()
	m := NewDashboardModel(sess, gw, "")

	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	dm, ok := model.(DashboardModel)
	if !ok {
		t.Fatalf("expected DashboardModel, got %T", model)
	}
	if dm.width != 100 || dm.height != 60 {
		t.Errorf("expected width=100, height=60, got width=%d, height=%d", dm.width, dm.height)
	}
}

func TestDashboardModel_CtrlCQuits(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-dash-quit",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw := gateway.NewMockGateway()
	m := NewDashboardModel(sess, gw, "")

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = model
	if cmd == nil {
		t.Error("expected quit command on Ctrl+C")
	}
}

func TestCheckCompletion(t *testing.T) {
	tests := []struct {
		name   string
		scores gateway.ConfidenceScores
		want   bool
	}{
		{"all 100", gateway.ConfidenceScores{100, 100, 100, 100}, true},
		{"all zero", gateway.ConfidenceScores{0, 0, 0, 0}, false},
		{"partial", gateway.ConfidenceScores{100, 80, 100, 100}, false},
		{"one not 100", gateway.ConfidenceScores{100, 100, 100, 99}, false},
		{"all above 100", gateway.ConfidenceScores{150, 120, 110, 105}, true},
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
