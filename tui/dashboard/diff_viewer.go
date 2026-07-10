package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/domain"
)

// handleDiffViewerUpdate processes keyboard/mouse events inside the diff viewer.
func (m DashboardModel) handleDiffViewerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.updateViewportSize()
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "q":
			// Cancel generation
			if m.cancelGen != nil {
				m.cancelGen()
			}
			m.showDiffViewer = false
			m.isWaitingDiffApproval = false
			return m, nil
		case "enter", "a":
			// Approve diffs!
			if m.diffApprovalChan != nil {
				close(m.diffApprovalChan)
				m.diffApprovalChan = nil
			}
			m.showDiffViewer = false
			m.isWaitingDiffApproval = false
			return m, nil
		case "tab":
			if len(m.proposedDiffs) > 0 {
				m.selectedDiffIdx = (m.selectedDiffIdx + 1) % len(m.proposedDiffs)
				m.updateDiffViewport()
			}
			return m, nil
		case "shift+tab":
			if len(m.proposedDiffs) > 0 {
				m.selectedDiffIdx = (m.selectedDiffIdx - 1 + len(m.proposedDiffs)) % len(m.proposedDiffs)
				m.updateDiffViewport()
			}
			return m, nil
		}

		if m.Settings.VimMode {
			switch keyMsg.String() {
			case "j":
				m.viewport.LineDown(1)
				return m, nil
			case "k":
				m.viewport.LineUp(1)
				return m, nil
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateDiffViewport loads the selected file's diff into the viewport.
func (m *DashboardModel) updateDiffViewport() {
	if len(m.proposedDiffs) == 0 || m.selectedDiffIdx < 0 || m.selectedDiffIdx >= len(m.proposedDiffs) {
		return
	}

	fd := m.proposedDiffs[m.selectedDiffIdx]
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Diff for File: %s (%d/%d)\n", fd.FileName, m.selectedDiffIdx+1, len(m.proposedDiffs)))
	builder.WriteString("Press [a] / [Enter] to Approve changes, [Esc] / [q] to Reject, [Tab] to Next File.\n\n")

	lines := strings.Split(fd.DiffText, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			builder.WriteString(fmt.Sprintf("+ %s\n", line[1:])) // High contrast prefix
		} else if strings.HasPrefix(line, "-") {
			builder.WriteString(fmt.Sprintf("- %s\n", line[1:])) // High contrast prefix
		} else {
			builder.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	m.viewport.SetContent(builder.String())
}

// startDiffApproval receives proposed diffs and enters diff approval state.
func (m DashboardModel) startDiffApproval(diffs []domain.FileDiff) (tea.Model, tea.Cmd) {
	m.proposedDiffs = diffs
	m.selectedDiffIdx = 0
	m.isWaitingDiffApproval = true
	m.showDiffViewer = true
	m.viewport = viewport.New(0, 0)
	m.updateViewportSize()
	m.updateDiffViewport()
	return m, nil
}
