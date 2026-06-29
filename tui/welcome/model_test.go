package welcome

import (
	"testing"
)

func TestNewWelcomeModel(t *testing.T) {
	m := NewWelcomeModel()
	if m.Phase != PhaseMenu {
		t.Errorf("expected PhaseMenu, got %v", m.Phase)
	}
	if len(m.Options) != 7 {
		t.Errorf("expected 7 menu options, got %d", len(m.Options))
	}
	if m.Action != ActionNone {
		t.Errorf("expected ActionNone, got %v", m.Action)
	}
}

func TestWelcomeModel_InitReturnsCmd(t *testing.T) {
	m := NewWelcomeModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}
