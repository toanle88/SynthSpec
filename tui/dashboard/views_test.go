package dashboard

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

func TestRenderHeader(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-header",
		Provider:    "mock",
		Scores:      gateway.ConfidenceScores{Functional: 50, Structural: 40, Security: 30, Compliance: 20},
	}
	m := NewDashboardModel(sess, gateway.NewMockGateway(), "")
	result := m.renderHeader()
	if result == "" {
		t.Error("expected non-empty header")
	}
}

func TestRenderFooter(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-footer",
		Provider:    "mock",
	}
	m := NewDashboardModel(sess, gateway.NewMockGateway(), "")
	result := m.renderFooter()
	if result == "" {
		t.Error("expected non-empty footer")
	}
}

func TestRenderThoughtBox(t *testing.T) {
	m := DashboardModel{
		Session: &state.Session{ProjectName: "test-thought"},
		Gateway: gateway.NewMockGateway(),
		ThoughtStreamState: ThoughtStreamState{
			streamingTokens: "thinking about the architecture...",
			isStreaming:     true,
		},
		width: 100,
	}
	result := m.renderThoughtBox()
	if !strings.Contains(result, "thinking about the architecture") {
		t.Errorf("expected thought content in output, got: %s", result)
	}
}

func TestRenderInterrogationState(t *testing.T) {
	sess := &state.Session{
		ProjectName:  "test-interro",
		Facts:        gateway.Facts{Functional: "test"},
		LastQuestion: "What do you think?",
		Scores:       gateway.ConfidenceScores{Functional: 50, Structural: 40, Security: 30, Compliance: 20},
	}
	m := NewDashboardModel(sess, gateway.NewMockGateway(), "")
	m.width = 80
	m.height = 40
	result := m.renderInterrogationState()
	if !strings.Contains(result, "What do you think?") {
		t.Errorf("expected question in output, got: %s", result)
	}
}

func TestGetFileGridPositions(t *testing.T) {
	m := DashboardModel{
		GenerationState: GenerationState{
			genFiles: []string{
				"01_domain_model_use_cases.md",
				"02_prd_functional.md",
				"03_system_architecture.md",
			},
		},
	}
	sourceIdx, grid := m.getFileGridPositions()
	if sourceIdx != 0 {
		t.Errorf("expected sourceIdx 0, got %d", sourceIdx)
	}
	if len(grid) == 0 {
		t.Fatal("expected non-empty grid")
	}
}

func TestGetGridPos(t *testing.T) {
	grid := [][]int{{1, 2, 3}, {4, 5, 6}}
	m := DashboardModel{
		GenerationState: GenerationState{
			genFiles: []string{"src.md", "a.md", "b.md", "c.md", "d.md", "e.md", "f.md"},
		},
	}
	sourceIdx := 0

	isSource, row, col := m.getGridPos(1, sourceIdx, grid)
	if isSource {
		t.Error("expected isSource=false for downstream file")
	}
	if row != 0 || col != 0 {
		t.Errorf("expected row=0, col=0, got row=%d, col=%d", row, col)
	}
}
