package dashboard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/tui/dashboard/keys"
)

func (m DashboardModel) handleUpdateKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	if m.isGenerating && !m.isWaitingApproval {
		keyStr := msg.String()
		if keyStr == "q" || keyStr == "esc" {
			if m.cancelGen != nil {
				m.cancelGen()
			}
			m.isGenerating = false
			m.genStatus = "Specification generation cancelled."
			return m, nil
		}
		if keyStr == "f" || keyStr == "F" {
			if m.forceFinishChan != nil {
				select {
				case <-m.forceFinishChan:
					// Already closed
				default:
					close(m.forceFinishChan)
				}
			}
			m.genStatus = "Force-finish requested. Saving current drafts..."
			return m, nil
		}
		return m, nil
	}
	if m.loading {
		return m, nil
	}
	return m.handleKeyMsg(msg)
}

// handleKeyMsg routes key presses to specific action handlers based on key type.
func (m DashboardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isWaitingApproval {
		return m.handleKeyMsgWaitingApproval(msg)
	}

	switch msg.Type {
	case tea.KeyCtrlK:
		if !m.isCompleted && !m.loading && !m.isGenerating && !m.showUpdatePrompt {
			return m.startOracleQuery("I do not know the answer. Please recommend the best compliance/architectural choice based on industry standards.")
		}
	case tea.KeyEnter:
		return m.handleKeyEnter()
	case tea.KeyUp:
		return m.handleKeyUp()
	case tea.KeyDown:
		return m.handleKeyDown()
	case tea.KeyPgUp, tea.KeyCtrlU:
		m.chatViewport.HalfPageUp()
		return m, nil
	case tea.KeyPgDown, tea.KeyCtrlD:
		m.chatViewport.HalfPageDown()
		return m, nil
	case tea.KeyLeft:
		return m.handleKeyLeft()
	case tea.KeyRight:
		return m.handleKeyRight()
	case tea.KeyEsc:
		return m.handleKeyEsc()
	case tea.KeyTab:
		return m.handleKeyTab(false)
	case tea.KeyShiftTab:
		return m.handleKeyTab(true)
	case tea.KeyRunes:
		return m.handleKeyRunes(msg)
	}
	return m, nil
}

// handleKeyTab cycles focus or selection in incomplete and completed phases.
func (m DashboardModel) handleKeyTab(shift bool) (tea.Model, tea.Cmd) {
	if m.showUpdatePrompt {
		return m, nil
	}
	if m.isCompleted {
		if shift {
			return m.navigateFilesUp()
		}
		return m.navigateFilesDown()
	}
	if !m.showTextInput {
		m.cycleChoice(shift)
	}
	return m, nil
}

func (m *DashboardModel) cycleChoice(shift bool) {
	choices := m.getChoicesList()
	if len(choices) == 0 {
		return
	}
	if shift {
		m.selectedChoiceIdx--
		if m.selectedChoiceIdx < 0 {
			m.selectedChoiceIdx = len(choices) - 1
		}
	} else {
		m.selectedChoiceIdx++
		if m.selectedChoiceIdx >= len(choices) {
			m.selectedChoiceIdx = 0
		}
	}
}

// getFileGridPositions returns the 2D grid mapping of indices in m.genFiles.
func (m DashboardModel) getFileGridPositions() (int, [][]int) {
	return keys.GetFileGridPositions(m.genFiles, domainModelFilename)
}

// getGridPos determines whether the selected index is the source file or is in the downstream grid.
func (m DashboardModel) getGridPos(selected int, sourceIdx int, grid [][]int) (bool, int, int) {
	return keys.GetGridPos(selected, sourceIdx, grid)
}

// navigateFilesUp moves selectedFileIdx upwards in the completed file list.
func (m DashboardModel) navigateFilesUp() (tea.Model, tea.Cmd) {
	m.selectedFileIdx = keys.NavigateUp(m.selectedFileIdx, m.genFiles, domainModelFilename)
	return m, nil
}

// navigateFilesDown moves selectedFileIdx downwards in the completed file list.
func (m DashboardModel) navigateFilesDown() (tea.Model, tea.Cmd) {
	m.selectedFileIdx = keys.NavigateDown(m.selectedFileIdx, m.genFiles, domainModelFilename)
	return m, nil
}

// navigateFilesLeft moves selectedFileIdx left in the grid.
func (m DashboardModel) navigateFilesLeft() (tea.Model, tea.Cmd) {
	m.selectedFileIdx = keys.NavigateLeft(m.selectedFileIdx, m.genFiles, domainModelFilename)
	return m, nil
}

// navigateFilesRight moves selectedFileIdx right in the grid.
func (m DashboardModel) navigateFilesRight() (tea.Model, tea.Cmd) {
	m.selectedFileIdx = keys.NavigateRight(m.selectedFileIdx, m.genFiles, domainModelFilename)
	return m, nil
}

// handleKeyMsgWaitingApproval processes key presses during the approval gate phase.
func (m DashboardModel) handleKeyMsgWaitingApproval(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "a", "enter":
		if m.approvalChan != nil {
			close(m.approvalChan)
			m.approvalChan = nil
		}
		m.isWaitingApproval = false
		m.genFileStatuses[domainModelFilename] = "done"
		m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
		return m, nil
	case "v":
		m.selectedFileIdx = 0
		return m.openFileViewer()
	case "e":
		return m.launchFileEditor(domainModelFilename)
	}
	return m, nil
}
