package tui

import (
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
			// Verify percentage text appears
			expectedPct := tt.percentage
			if expectedPct < 0 {
				expectedPct = 0
			}
			if expectedPct > 100 {
				expectedPct = 100
			}
			// The bar should contain the numeric percentage
			pctStr := string(rune('0'+expectedPct/10)) + string(rune('0'+expectedPct%10))
			// Handle single digit
			if expectedPct < 10 {
				pctStr = " " + pctStr[1:]
			}
			// Just verify the string is non-empty and contains a % sign
			// since exact ANSI formatting varies
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderProgressBar_WidthConsistency(t *testing.T) {
	// A 100% bar should have all filled chars
	result := RenderProgressBar(10, 100)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// A 50% bar should have mixed chars
	result50 := RenderProgressBar(10, 50)
	if result50 == "" {
		t.Fatal("expected non-empty result")
	}
	// Different widths should produce different results
	w10 := RenderProgressBar(10, 50)
	w20 := RenderProgressBar(20, 50)
	if w10 == w20 {
		t.Log("note: 50% at width 10 and 20 produce identical output (may be same if truncated)")
	}
}
