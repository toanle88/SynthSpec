package welcome

import (
	"testing"
)

func TestWelcomeModel_ViewReturnsContent(t *testing.T) {
	m := NewWelcomeModel()
	view := m.View()
	if view == "" {
		t.Error("expected non-empty View")
	}
}
