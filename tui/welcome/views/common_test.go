package views

import (
	"strings"
	"testing"
)

func TestRenderLogo(t *testing.T) {
	logo := RenderLogo()
	if !strings.Contains(logo, "_____") {
		t.Errorf("expected logo to contain horizontal bars, got %q", logo)
	}
}
