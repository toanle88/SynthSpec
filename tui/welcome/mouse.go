package welcome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui/shared"
)

func (m WelcomeModel) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		return m.handleMouseWheelUp(), nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return m.handleMouseWheelDown(), nil
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.handleMouseLeftClick(msg)
	}
	return m, nil
}

func (m WelcomeModel) handleMouseWheelUp() WelcomeModel {
	switch m.Phase {
	case PhaseMenu:
		m.SelectedOption = shared.MaxInt(0, m.SelectedOption-1)
	case PhaseBlueprintSelect:
		m.SelectedBPIdx = shared.MaxInt(0, m.SelectedBPIdx-1)
	case PhaseResumeSelect:
		m.SelectedProject = shared.MaxInt(0, m.SelectedProject-1)
	case PhaseExportSelect:
		m.SelectedProject = shared.MaxInt(0, m.SelectedProject-1)
	case PhaseProjectMenu:
		m.SelectedProjectOption = shared.MaxInt(0, m.SelectedProjectOption-1)
	case PhaseProjectViewFiles:
		m.SelectedProjectFile = shared.MaxInt(0, m.SelectedProjectFile-1)
	case PhaseFileContentViewer:
		m.ViewerScrollOffset = shared.MaxInt(0, m.ViewerScrollOffset-1)
	case PhaseSettings:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Blur()
		}
		m.SelectedSettingIdx = shared.MaxInt(0, m.SelectedSettingIdx-1)
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Focus()
		}
	}
	return m
}

func (m WelcomeModel) handleMouseWheelDown() WelcomeModel {
	switch m.Phase {
	case PhaseMenu:
		m.SelectedOption = shared.MinInt(len(m.Options)-1, m.SelectedOption+1)
	case PhaseBlueprintSelect:
		m.SelectedBPIdx = shared.MinInt(len(m.Blueprints), m.SelectedBPIdx+1)
	case PhaseResumeSelect:
		m.SelectedProject = shared.MinInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case PhaseExportSelect:
		m.SelectedProject = shared.MinInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case PhaseProjectMenu:
		m.SelectedProjectOption = shared.MinInt(len(m.ProjectOptions)-1, m.SelectedProjectOption+1)
	case PhaseProjectViewFiles:
		m.SelectedProjectFile = shared.MinInt(len(m.ProjectFiles)-1, m.SelectedProjectFile+1)
	case PhaseFileContentViewer:
		pageSize := 10
		if m.height > 12 {
			pageSize = m.height - 12
		}
		maxScroll := shared.MaxInt(0, len(m.ViewerLines)-pageSize)
		m.ViewerScrollOffset = shared.MinInt(maxScroll, m.ViewerScrollOffset+1)
	case PhaseSettings:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Blur()
		}
		m.SelectedSettingIdx = shared.MinInt(4, m.SelectedSettingIdx+1)
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Focus()
		}
	}
	return m
}

func (m WelcomeModel) handleMouseLeftClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	rendered := shared.StripANSI(m.View())
	lines := strings.Split(rendered, "\n")
	if msg.Y < 0 || msg.Y >= len(lines) {
		return m, nil
	}
	line := lines[msg.Y]
	switch m.Phase {
	case PhaseMenu:
		return m.handleMenuClick(line)
	case PhaseBlueprintSelect:
		return m.handleBlueprintClick(line)
	case PhaseResumeSelect:
		return m.handleResumeClick(line)
	case PhaseExportSelect:
		return m.handleExportClick(line)
	case PhaseProjectMenu:
		return m.handleProjectMenuClick(line)
	case PhaseProjectViewFiles:
		return m.handleProjectViewFilesClick(line)
	case PhaseDeleteConfirm:
		return m.handleDeleteConfirmClick(line)
	case PhaseSettings:
		return m.handleSettingsClick(line)
	}
	return m, nil
}

func (m WelcomeModel) handleMenuClick(line string) (tea.Model, tea.Cmd) {
	for i, opt := range m.Options {
		if strings.Contains(line, opt) {
			m.SelectedOption = i
			m.handleMenuSelection()
			return m, nil
		}
	}
	return m, nil
}

func (m WelcomeModel) handleBlueprintClick(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "None (Start from scratch)") || strings.Contains(line, "Start with an empty specification session") {
		m.SelectedBPIdx = 0
		m.SelectedBlueprint = ""
		m.IsNewProject = true
		m.SelectedProjectOption = 0
		m.Phase = PhaseProjectMenu
		return m, nil
	}
	for i, bp := range m.Blueprints {
		if strings.Contains(line, bp.Name) || (bp.Description != "" && strings.Contains(line, bp.Description)) {
			m.SelectedBPIdx = i + 1
			m.SelectedBlueprint = bp.ID
			m.IsNewProject = true
			m.SelectedProjectOption = 0
			m.Phase = PhaseProjectMenu
			return m, nil
		}
	}
	return m, nil
}

func (m WelcomeModel) handleResumeClick(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "Search:") {
		m.filterInput.Focus()
		return m, nil
	}
	for i, proj := range m.FilteredProjects {
		if strings.Contains(line, proj) {
			m.SelectedProject = i
			m.ProjectName = proj
			m.IsNewProject = false
			m.SelectedProjectOption = 0
			m.Phase = PhaseProjectMenu
			return m, nil
		}
	}
	return m, nil
}

func (m WelcomeModel) handleExportClick(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "Search:") {
		m.filterInput.Focus()
		return m, nil
	}
	for i, proj := range m.FilteredProjects {
		if strings.Contains(line, proj) {
			m.SelectedProject = i
			m.ProjectName = proj
			m.IsNewProject = false
			m.SelectedProjectOption = 2
			m.Phase = PhaseProjectMenu
			return m, nil
		}
	}
	return m, nil
}

func (m WelcomeModel) handleProjectMenuClick(line string) (tea.Model, tea.Cmd) {
	for i, opt := range m.ProjectOptions {
		if strings.Contains(line, opt) {
			m.SelectedProjectOption = i
			return m.handleProjectMenuSelection()
		}
	}
	return m, nil
}

func (m WelcomeModel) handleProjectViewFilesClick(line string) (tea.Model, tea.Cmd) {
	for i, fName := range m.ProjectFiles {
		if strings.Contains(line, fName) {
			m.SelectedProjectFile = i
			filePath := filepath.Join(state.GetSessionDir(m.ProjectName), "output", fName)
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				m.alertTitle = "Error Reading File"
				m.alertMessage = fmt.Sprintf("Could not read file: %v", err)
				m.alertNext = PhaseProjectViewFiles
				m.Phase = PhaseStatusAlert
				return m, nil
			}
			highlighted := shared.HighlightMarkdown(string(contentBytes))
			m.ViewerLines = strings.Split(highlighted, "\n")
			m.ViewerScrollOffset = 0
			m.Phase = PhaseFileContentViewer
			return m, nil
		}
	}
	return m, nil
}

func (m WelcomeModel) handleDeleteConfirmClick(line string) (tea.Model, tea.Cmd) {
	if strings.Contains(line, "[ Yes, Delete ]") {
		return m.updateDeleteConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	}
	if strings.Contains(line, cancelLiteral) {
		return m.updateDeleteConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	}
	return m, nil
}

func (m WelcomeModel) handleSettingsClick(line string) (tea.Model, tea.Cmd) {
	settingFields := []string{"API Timeout (seconds)", "Max API Retries", "Default Output Folder", "Debug Logging (opt-in)", "Vim Keybindings (hjkl)"}
	for i, field := range settingFields {
		if strings.Contains(line, field) {
			if m.SelectedSettingIdx < len(m.settingInputs) {
				m.settingInputs[m.SelectedSettingIdx].Blur()
			}
			m.SelectedSettingIdx = i
			if i < len(m.settingInputs) {
				m.settingInputs[i].Focus()
			}
			switch i {
			case 3:
				m.Settings.Debug = !m.Settings.Debug
				logger.LogEvent("TUI", fmt.Sprintf("Debug logging toggled via click: %t", m.Settings.Debug))
			case 4:
				m.Settings.VimMode = !m.Settings.VimMode
				logger.LogEvent("TUI", fmt.Sprintf("Vim mode toggled via click: %t", m.Settings.VimMode))
			}
			return m, nil
		}
	}
	if strings.Contains(line, "[ Save Settings ]") {
		m.saveSettingsFromInputs()
		return m, nil
	}
	if strings.Contains(line, cancelLiteral) {
		m.Phase = PhaseMenu
		return m, nil
	}
	return m, nil
}
