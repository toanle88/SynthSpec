package dashboard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/state"
)

// handleKeyEnter processes Enter key presses, submitting inputs, selecting options, or launching full editors.
func (m DashboardModel) handleKeyEnter() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m.handleKeyEnterUpdatePrompt()
	}
	if m.isCompleted {
		return m.handleKeyEnterCompleted()
	}
	if m.showTextInput {
		return m.handleKeyEnterTextInput()
	}
	return m.handleKeyEnterChoiceSelection()
}

// handleKeyEnterUpdatePrompt processes Enter key presses inside the manual requirements update prompt.
func (m DashboardModel) handleKeyEnterUpdatePrompt() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.updateInput.Value())
	if val == "" {
		return m, nil
	}
	m.updateInput.SetValue("")
	m.showUpdatePrompt = false
	m.isCompleted = false
	return m.startOracleQuery("I have a new requirement/change: " + val)
}

// handleKeyEnterCompleted processes Enter key presses on the completion screen, opening the document viewer.
func (m DashboardModel) handleKeyEnterCompleted() (tea.Model, tea.Cmd) {
	if m.showViewer {
		return m, nil
	}
	if len(m.genFiles) > 0 {
		return m.openFileViewer()
	}
	return m, nil
}

// handleKeyEnterTextInput processes Enter key presses when typing answers directly into the console.
func (m DashboardModel) handleKeyEnterTextInput() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.textInput.Value())
	if val == "" {
		return m, nil
	}

	m.textInput.SetValue("")

	if val == ":edit" {
		editorCmd, tempPath, err := state.GetEditorCommand(m.Session.GetProjectName(), m.Session.GetFacts())
		if err != nil {
			m.setError(err)
			return m, nil
		}
		m.editorTempPath = tempPath
		return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	}

	if val == ":override" || val == ":bypass" {
		scores := domain.ConfidenceScores{
			Functional: 100,
			Structural: 100,
			Security:   100,
			Compliance: 100,
		}
		rationales := domain.DimensionRationales{
			Functional: "Bypassed by operator",
			Structural: "Bypassed by operator",
			Security:   "Bypassed by operator",
			Compliance: "Bypassed by operator",
		}
		_ = m.Session.UpdateScores(scores, rationales)
		m.isCompleted = true
		return m.triggerRegeneration()
	}

	if val == ":undo" {
		if err := m.Session.Undo(); err == nil {
			m.isCompleted = checkCompletion(m.Session.GetScores())
			m.showTextInput = len(m.Session.GetLastChoices()) == 0
			m.err = nil
			return m, func() tea.Msg { return initQueryMsg{} }
		} else {
			m.setError(err)
			return m, nil
		}
	}

	return m.startOracleQuery(val)
}

// handleKeyEnterChoiceSelection processes Enter key presses when selecting choices in the list.
func (m DashboardModel) handleKeyEnterChoiceSelection() (tea.Model, tea.Cmd) {
	choices := m.getChoicesList()
	selected := choices[m.selectedChoiceIdx]

	if selected == "Custom user input..." {
		m.showTextInput = true
		m.textInput.Focus()
		m.textInput.SetValue("")
		return m, nil
	}

	var val string
	if selected == "Let AI decide" {
		val = "Let the AI decide based on current facts and context."
	} else {
		val = m.Session.GetLastChoices()[m.selectedChoiceIdx]
	}

	return m.startOracleQuery(val)
}

// handleKeyUp navigates upwards through choices in the interactive list.
func (m DashboardModel) handleKeyUp() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesUp()
	}
	if !m.showTextInput {
		choices := m.getChoicesList()
		m.selectedChoiceIdx--
		if m.selectedChoiceIdx < 0 {
			m.selectedChoiceIdx = len(choices) - 1
		}
	}
	return m, nil
}

// handleKeyDown navigates downwards through choices in the interactive list.
func (m DashboardModel) handleKeyDown() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesDown()
	}
	if !m.showTextInput {
		choices := m.getChoicesList()
		m.selectedChoiceIdx++
		if m.selectedChoiceIdx >= len(choices) {
			m.selectedChoiceIdx = 0
		}
	}
	return m, nil
}

// handleKeyLeft navigates leftwards in the file selection layout.
func (m DashboardModel) handleKeyLeft() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesLeft()
	}
	return m, nil
}

// handleKeyRight navigates rightwards in the file selection layout.
func (m DashboardModel) handleKeyRight() (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.navigateFilesRight()
	}
	return m, nil
}

// handleKeyEsc dismisses active errors, update prompts, or custom text inputs.
func (m DashboardModel) handleKeyEsc() (tea.Model, tea.Cmd) {
	if m.err != nil {
		m.err = nil
		return m, nil
	}
	if m.showUpdatePrompt {
		m.showUpdatePrompt = false
		m.updateInput.Blur()
		if m.isCLIUpdateMode {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.showTextInput && len(m.Session.GetLastChoices()) > 0 {
		m.showTextInput = false
		m.textInput.Blur()
	}
	return m, nil
}

// handleKeyRunes routes rune character keys based on session completion status.
func (m DashboardModel) handleKeyRunes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		return m.handleKeyRunesCompleted(msg)
	}
	return m.handleKeyRunesIncomplete(msg)
}

// handleKeyRunesCompleted handles shortcut keys (generate, edit, update, quit) on the final completion dashboard screen.
func (m DashboardModel) handleKeyRunesCompleted(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch strings.ToLower(key) {
	case "g":
		return m.triggerRegeneration()
	case "e":
		return m.launchExternalEditor()
	case "u":
		return m.activateUpdatePrompt()
	case "v":
		if len(m.genFiles) > 0 {
			return m.openFileViewer()
		}
	case "k":
		if m.Settings.VimMode {
			return m.navigateFilesUp()
		}
	case "j":
		if m.Settings.VimMode {
			return m.navigateFilesDown()
		}
	case "h":
		if m.Settings.VimMode {
			return m.navigateFilesLeft()
		}
	case "l":
		if m.Settings.VimMode {
			return m.navigateFilesRight()
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// handleKeyRunesIncomplete processes vi-style navigation keys (j/k) when the requirements phase is still ongoing.
func (m DashboardModel) handleKeyRunesIncomplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showTextInput {
		return m, nil
	}
	key := msg.String()
	switch strings.ToLower(key) {
	case "g":
		return m.triggerRegeneration()
	case "e":
		return m.launchExternalEditor()
	case "u":
		return m.activateUpdatePrompt()
	case "q":
		return m, tea.Quit
	case "k":
		if m.Settings.VimMode {
			choices := m.getChoicesList()
			m.selectedChoiceIdx--
			if m.selectedChoiceIdx < 0 {
				m.selectedChoiceIdx = len(choices) - 1
			}
		}
	case "j":
		if m.Settings.VimMode {
			choices := m.getChoicesList()
			m.selectedChoiceIdx++
			if m.selectedChoiceIdx >= len(choices) {
				m.selectedChoiceIdx = 0
			}
		}
	}
	return m, nil
}
