package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui/shared"
)

// handleViewerUpdate processes updates to the document viewer overlay, handling size changes and dismissal key events.
func (m DashboardModel) handleViewerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.updateViewportSize()
	}
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return m.handleViewerMouseUpdate(mouseMsg)
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleViewerKeyUpdate(keyMsg)
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DashboardModel) handleViewerMouseUpdate(mouseMsg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if mouseMsg.Button == tea.MouseButtonWheelUp || mouseMsg.Button == tea.MouseButtonWheelDown {
		m.viewport, cmd = m.viewport.Update(mouseMsg)
		return m, cmd
	}
	if mouseMsg.Action == tea.MouseActionPress && mouseMsg.Button == tea.MouseButtonLeft {
		rendered := shared.StripANSI(m.View())
		lines := strings.Split(rendered, "\n")
		if mouseMsg.Y >= 0 && mouseMsg.Y < len(lines) {
			return m.handleViewerLeftClick(lines[mouseMsg.Y])
		}
	}
	return m, nil
}

func (m DashboardModel) handleViewerLeftClick(line string) (tea.Model, tea.Cmd) {
	for i, file := range m.genFiles {
		if strings.Contains(line, file) {
			m.selectedFileIdx = i
			return m.openFileViewer()
		}
	}
	if strings.Contains(line, "Back") {
		m.showViewer = false
		return m, nil
	}
	if strings.Contains(line, "Toggle Layout") {
		m.isFullScreenViewer = !m.isFullScreenViewer
		m.updateViewportSize()
		return m.openFileViewer()
	}
	if m.isWaitingApproval {
		if strings.Contains(line, "Approve") {
			m.showViewer = false
			if m.approvalChan != nil {
				close(m.approvalChan)
				m.approvalChan = nil
			}
			m.isWaitingApproval = false
			m.genFileStatuses[domainModelFilename] = "done"
			m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
			return m, nil
		}
		if strings.Contains(line, "Edit") {
			return m.launchFileEditor(domainModelFilename)
		}
	}
	return m, nil
}

func (m DashboardModel) handleViewerKeyUpdate(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keyMsg.Type == tea.KeyEsc || keyMsg.String() == "q" {
		m.showViewer = false
		return m, nil
	}
	if keyMsg.String() == "f" || keyMsg.String() == "F" {
		m.isFullScreenViewer = !m.isFullScreenViewer
		m.updateViewportSize()
		return m, nil
	}
	if m.isWaitingApproval {
		switch strings.ToLower(keyMsg.String()) {
		case "a", "enter":
			m.showViewer = false
			if m.approvalChan != nil {
				close(m.approvalChan)
				m.approvalChan = nil
			}
			m.isWaitingApproval = false
			m.genFileStatuses[domainModelFilename] = "done"
			m.genStatus = "Domain Model approved! Commencing downstream parallel generation..."
			return m, nil
		case "e":
			return m.launchFileEditor(domainModelFilename)
		}
	}
	if m.Settings.VimMode {
		switch keyMsg.String() {
		case "j":
			m.viewport.LineDown(1)
			return m, nil
		case "k":
			m.viewport.LineUp(1)
			return m, nil
		case "d", "ctrl+d":
			m.viewport.HalfPageDown()
			return m, nil
		case "u", "ctrl+u":
			m.viewport.HalfPageUp()
			return m, nil
		}
	}
	return m, nil
}

// updateViewportSize computes the width and height of the viewer viewport depending on layout mode.
func (m *DashboardModel) updateViewportSize() {
	width := m.width - 4
	if !m.isFullScreenViewer {
		sidebarWidth := 34
		chatWidth := m.width - sidebarWidth - 8
		if chatWidth < 40 {
			chatWidth = 40
		}
		width = chatWidth - 4
	}
	m.viewport.Width = width
	m.viewport.Height = m.height - 6
}

// openFileViewer opens the full screen Markdown document viewer viewport overlay.
func (m DashboardModel) openFileViewer() (tea.Model, tea.Cmd) {
	var selectedFile string
	if m.isWaitingApproval {
		selectedFile = domainModelFilename
	} else if len(m.genFiles) > 0 && m.selectedFileIdx >= 0 && m.selectedFileIdx < len(m.genFiles) {
		selectedFile = m.genFiles[m.selectedFileIdx]
	} else {
		selectedFile = domainModelFilename
	}
	dir := m.OutputDir
	if dir == "" {
		dir = filepath.Join(state.GetSessionDir(m.Session.ProjectName), "output")
	}
	filePath := filepath.Join(dir, selectedFile)
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		m.setError(fmt.Errorf("failed to read file %s: %w", selectedFile, err))
		return m, nil
	}
	content := shared.HighlightMarkdown(string(contentBytes))

	m.viewport = viewport.New(0, 0)
	m.updateViewportSize()
	m.viewport.SetContent(content)
	m.showViewer = true
	return m, nil
}
