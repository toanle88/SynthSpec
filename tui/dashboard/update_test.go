package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

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
