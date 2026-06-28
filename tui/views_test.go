package tui

import (
	"strings"
	"testing"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

func TestWrapSingleLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
		want  int // expected number of result lines
	}{
		{"shorter than width", "hello", 20, 1},
		{"exact width", "hello world", 11, 1},
		{"longer than width", "hello world foo bar baz", 10, 3},
		{"single word longer than width", "superlongword", 5, 1}, // just returns it as-is
		{"empty string", "", 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapSingleLine(tt.line, tt.width)
			if len(result) != tt.want {
				t.Errorf("expected %d lines, got %d: %v", tt.want, len(result), result)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	input := "this is a test of the text wrapping function it should split properly"
	result := wrapText(input, 15)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("expected multiple lines wrapped at width 15, got %d: %s", len(lines), result)
	}
}

func TestWrapText_Empty(t *testing.T) {
	result := wrapText("", 10)
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestWrapText_WidthZero(t *testing.T) {
	input := "some text"
	result := wrapText(input, 0)
	if result != input {
		t.Errorf("expected original text with width 0, got %q", result)
	}
}

func TestRenderHeader(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-header",
		Provider:    "mock",
		Scores:      gateway.ConfidenceScores{50, 40, 30, 20},
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
		Session:         &state.Session{ProjectName: "test-thought"},
		Gateway:         gateway.NewMockGateway(),
		streamingTokens: "thinking about the architecture...",
		isStreaming:     true,
		width:           100,
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
		Scores:       gateway.ConfidenceScores{50, 40, 30, 20},
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
		genFiles: []string{
			"01_domain_model_use_cases.md",
			"02_prd_functional.md",
			"03_system_architecture.md",
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
	m := DashboardModel{genFiles: []string{"src.md", "a.md", "b.md", "c.md", "d.md", "e.md", "f.md"}}
	sourceIdx := 0

	// First downstream file (index 1) -> row 0, col 0
	isSource, row, col := m.getGridPos(1, sourceIdx, grid)
	if isSource {
		t.Error("expected isSource=false for downstream file")
	}
	if row != 0 || col != 0 {
		t.Errorf("expected row=0, col=0, got row=%d, col=%d", row, col)
	}

	// Source file (index 0) -> isSource=true
	isSource, _, _ = m.getGridPos(0, sourceIdx, grid)
	if !isSource {
		t.Error("expected isSource=true for source file")
	}
}
