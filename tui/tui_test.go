package tui

import (
	"testing"
)

func TestNewWelcomeModel(t *testing.T) {
	m := NewWelcomeModel()
	if m.ProjectName != "" {
		t.Errorf("expected default project name to be empty, got %q", m.ProjectName)
	}
}
