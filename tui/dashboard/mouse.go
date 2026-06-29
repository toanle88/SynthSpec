package dashboard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/tui/shared"
)

func (m DashboardModel) handleUpdateMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		if !m.loading {
			m.chatViewport.LineUp(3)
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if !m.loading {
			m.chatViewport.LineDown(3)
		}
		return m, nil
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.handleMouseLeftClickDashboard(msg)
	}
	return m, nil
}

func (m DashboardModel) handleMouseLeftClickDashboard(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	rendered := shared.StripANSI(m.View())
	lines := strings.Split(rendered, "\n")
	if msg.Y < 0 || msg.Y >= len(lines) {
		return m, nil
	}
	line := lines[msg.Y]
	if m.isCompleted || m.isGenerating {
		return m.handleMouseLeftClickGeneratingCompleted(line)
	}
	if !m.isCompleted && !m.loading && !m.showTextInput {
		return m.handleMouseLeftClickChoices(line)
	}
	if m.showTextInput && strings.Contains(line, "> ") {
		m.textInput.Focus()
		return m, nil
	}
	if m.showUpdatePrompt && strings.Contains(line, "> ") {
		m.updateInput.Focus()
		return m, nil
	}
	return m, nil
}

func (m DashboardModel) handleMouseLeftClickGeneratingCompleted(line string) (tea.Model, tea.Cmd) {
	for i, file := range m.genFiles {
		if strings.Contains(line, file) {
			m.selectedFileIdx = i
			return m.openFileViewer()
		}
	}
	if m.isCompleted {
		return m.handleMouseLeftClickCompleted(line)
	}
	if m.isWaitingApproval {
		return m.handleMouseLeftClickWaitingApproval(line)
	}
	return m, nil
}

func (m DashboardModel) handleMouseLeftClickCompleted(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "Regenerate files") {
		return m.triggerRegeneration()
	}
	if strings.Contains(line, "Add new requirements") {
		return m.activateUpdatePrompt()
	}
	if strings.Contains(line, "launch Editor") {
		return m.launchExternalEditor()
	}
	if strings.Contains(line, "Save & Exit CLI") {
		return m, tea.Quit
	}
	return m, nil
}

func (m DashboardModel) handleMouseLeftClickWaitingApproval(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "View "+domainModelFilename) {
		m.selectedFileIdx = 0
		return m.openFileViewer()
	}
	if strings.Contains(line, "Edit "+domainModelFilename) {
		return m.launchFileEditor(domainModelFilename)
	}
	if strings.Contains(line, "Approve and Resume") {
		if m.approvalChan != nil {
			close(m.approvalChan)
			m.approvalChan = nil
		}
		m.isWaitingApproval = false
		m.genFileStatuses[domainModelFilename] = "done"
		m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
		return m, nil
	}
	return m, nil
}

func (m DashboardModel) handleMouseLeftClickChoices(line string) (tea.Model, tea.Cmd) {
	choices := m.getChoicesList()
	for i, choice := range choices {
		if strings.Contains(line, choice) {
			m.selectedChoiceIdx = i
			return m.handleKeyEnterChoiceSelection()
		}
	}
	return m, nil
}
