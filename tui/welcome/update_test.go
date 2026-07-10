package welcome

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/state"
)

func TestWelcomeModel_CtrlCQuits(t *testing.T) {
	m := NewWelcomeModel()
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit command on Ctrl+C")
	}
	wm, ok := model.(WelcomeModel)
	if !ok {
		t.Fatalf("expected WelcomeModel, got %T", model)
	}
	if wm.Action != ActionExit {
		t.Errorf("expected ActionExit, got %v", wm.Action)
	}
}

func TestWelcomeModel_KeyNavigation(t *testing.T) {
	m := NewWelcomeModel()

	// Down arrow
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	wm := model.(WelcomeModel)
	if wm.SelectedOption != 1 {
		t.Errorf("expected SelectedOption=1 after down, got %d", wm.SelectedOption)
	}

	// Up arrow
	model, _ = wm.Update(tea.KeyMsg{Type: tea.KeyUp})
	wm = model.(WelcomeModel)
	if wm.SelectedOption != 0 {
		t.Errorf("expected SelectedOption=0 after up, got %d", wm.SelectedOption)
	}

	// Tab (should wrap around)
	model, _ = wm.Update(tea.KeyMsg{Type: tea.KeyTab})
	wm = model.(WelcomeModel)
	if wm.SelectedOption != 1 {
		t.Errorf("expected SelectedOption=1 after tab, got %d", wm.SelectedOption)
	}

	// Shift+Tab (should go backward)
	model, _ = wm.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	wm = model.(WelcomeModel)
	if wm.SelectedOption != 0 {
		t.Errorf("expected SelectedOption=0 after shift+tab, got %d", wm.SelectedOption)
	}
}

func TestWelcomeModel_WindowSize(t *testing.T) {
	m := NewWelcomeModel()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	wm := model.(WelcomeModel)
	if wm.width != 100 || wm.height != 50 {
		t.Errorf("expected width=100, height=50, got width=%d, height=%d", wm.width, wm.height)
	}
}

func TestWelcomeModel_EnterOnCreate(t *testing.T) {
	m := NewWelcomeModel()
	// Select "Create New Project" (option 0) and press enter
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm := model.(WelcomeModel)
	// Should transition to PhaseCreateInput
	if wm.Phase != PhaseCreateInput {
		t.Errorf("expected PhaseCreateInput after enter on Create New Project, got %v", wm.Phase)
	}
}

func TestWelcomeModel_EnterOnResume(t *testing.T) {
	// Create a project first so ListProjects finds it
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	sess := &state.Session{
		ProjectName: "test-resume-proj",
		Provider:    "mock",
		Model:       "mock-model",
	}
	sess.Save()

	m := NewWelcomeModel()
	// Navigate to option 1 (Resume) and press enter
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	wm := model.(WelcomeModel)
	model, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm = model.(WelcomeModel)
	if wm.Phase != PhaseResumeSelect {
		t.Errorf("expected PhaseResumeSelect, got %v", wm.Phase)
	}
}

func TestWelcomeModel_TypeProjectName(t *testing.T) {
	m := NewWelcomeModel()
	// Enter PhaseCreateInput by pressing Enter on option 0
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm := model.(WelcomeModel)

	if wm.Phase != PhaseCreateInput {
		t.Fatalf("expected PhaseCreateInput, got %v", wm.Phase)
	}

	if !wm.textInput.Focused() {
		t.Fatal("expected textInput to be focused")
	}

	// Send key 'a'
	model, _ = wm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	wm = model.(WelcomeModel)

	if wm.textInput.Value() != "a" {
		t.Errorf("expected textInput value 'a', got %q", wm.textInput.Value())
	}
}

