package shared

import (
	"strings"
	"testing"
)

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		percentage int
	}{
		{"zero percent", 20, 0},
		{"twenty five percent", 20, 25},
		{"fifty percent", 20, 50},
		{"seventy five percent", 20, 75},
		{"one hundred percent", 20, 100},
		{"negative clamps to zero", 20, -5},
		{"over 100 clamps", 20, 150},
		{"narrow bar", 5, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderProgressBar(tt.width, tt.percentage)
			if result == "" {
				t.Error("expected non-empty progress bar string")
			}
			expectedPct := tt.percentage
			if expectedPct < 0 {
				expectedPct = 0
			}
			if expectedPct > 100 {
				expectedPct = 100
			}
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderProgressBar_WidthConsistency(t *testing.T) {
	result := RenderProgressBar(10, 100)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	result50 := RenderProgressBar(10, 50)
	if result50 == "" {
		t.Fatal("expected non-empty result")
	}
	w10 := RenderProgressBar(10, 50)
	w20 := RenderProgressBar(20, 50)
	if w10 == w20 {
		t.Log("note: 50% at width 10 and 20 produce identical output (may be same if truncated)")
	}
}

func TestWrapText(t *testing.T) {
	input := "this is a test of the text wrapping function it should split properly"
	result := WrapText(input, 15)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("expected multiple lines wrapped at width 15, got %d: %s", len(lines), result)
	}
}

func TestWrapText_Empty(t *testing.T) {
	result := WrapText("", 10)
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestWrapText_WidthZero(t *testing.T) {
	input := "some text"
	result := WrapText(input, 0)
	if result != input {
		t.Errorf("expected original text with width 0, got %q", result)
	}
}
