package welcome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui/shared"
)

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.Action = ActionExit
			return m, tea.Quit
		}
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	}

	return m, nil
}

func (m WelcomeModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Phase {
	case PhaseMenu:
		return m.updateMenu(msg)
	case PhaseCreateInput:
		return m.updateCreateInput(msg)
	case PhaseBlueprintSelect:
		return m.updateBlueprintSelect(msg)
	case PhaseResumeSelect:
		return m.updateResumeSelect(msg)
	case PhaseExportSelect:
		return m.updateExportSelect(msg)
	case PhaseProjectMenu:
		return m.updateProjectMenu(msg)
	case PhaseProjectViewFiles:
		return m.updateProjectViewFiles(msg)
	case PhaseFileContentViewer:
		return m.updateFileContentViewer(msg)
	case PhaseDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case PhaseStatusAlert:
		if m.alertNext != PhaseMenu {
			m.Phase = m.alertNext
			m.alertNext = PhaseMenu
		} else {
			m.Phase = PhaseMenu
		}
		return m, nil
	case PhaseSettings:
		return m.updateSettings(msg)
	case PhaseViewAssets, PhaseAuditWorkspace:
		if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
			m.Phase = PhaseMenu
		}
	}
	return m, nil
}

func navigateList(currentVal, length int, vimMode bool, msg tea.KeyMsg) int {
	switch msg.String() {
	case "up", keyCtrlP:
		return shared.MaxInt(0, currentVal-1)
	case "k":
		if vimMode {
			return shared.MaxInt(0, currentVal-1)
		}
	case "down", keyCtrlN:
		return shared.MinInt(length-1, currentVal+1)
	case "j":
		if vimMode {
			return shared.MinInt(length-1, currentVal+1)
		}
	case "tab":
		if length > 0 {
			return (currentVal + 1) % length
		}
	case keyShiftTab:
		if length > 0 {
			return (currentVal - 1 + length) % length
		}
	}
	return currentVal
}

func (m WelcomeModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedOption = navigateList(m.SelectedOption, len(m.Options), m.Settings.VimMode, msg)
	case "enter":
		m.handleMenuSelection()
	case "q", "esc":
		m.Action = ActionExit
		return m, tea.Quit
	}
	return m, nil
}

func (m WelcomeModel) updateCreateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.textInput.Value())
		if name != "" {
			m.ProjectName = name
			m.Phase = PhaseBlueprintSelect
			m.SelectedBPIdx = 0
			return m, nil
		}
	case tea.KeyEsc:
		m.Phase = PhaseMenu
	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m WelcomeModel) updateBlueprintSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedBPIdx = navigateList(m.SelectedBPIdx, len(m.Blueprints)+1, m.Settings.VimMode, msg)
	case "enter":
		if m.SelectedBPIdx == 0 {
			m.SelectedBlueprint = ""
		} else {
			m.SelectedBlueprint = m.Blueprints[m.SelectedBPIdx-1].ID
		}
		m.IsNewProject = true
		m.SelectedProjectOption = 0
		m.Phase = PhaseProjectMenu
		return m, nil
	case "esc":
		m.Phase = PhaseCreateInput
	}
	return m, nil
}

func (m WelcomeModel) updateResumeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedProject = navigateList(m.SelectedProject, len(m.FilteredProjects), m.Settings.VimMode, msg)
	case "enter":
		if len(m.FilteredProjects) > 0 && m.SelectedProject >= 0 && m.SelectedProject < len(m.FilteredProjects) {
			m.ProjectName = m.FilteredProjects[m.SelectedProject]
			m.IsNewProject = false
			m.SelectedProjectOption = 0
			m.Phase = PhaseProjectMenu
			return m, nil
		}
	case "esc":
		m.Phase = PhaseMenu
	default:
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.runFuzzyFiltering()
	}
	return m, cmd
}

func (m WelcomeModel) updateExportSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedProject = navigateList(m.SelectedProject, len(m.FilteredProjects), m.Settings.VimMode, msg)
	case "enter":
		if len(m.FilteredProjects) > 0 && m.SelectedProject >= 0 && m.SelectedProject < len(m.FilteredProjects) {
			m.ProjectName = m.FilteredProjects[m.SelectedProject]
			m.IsNewProject = false
			m.SelectedProjectOption = 2
			m.Phase = PhaseProjectMenu
			return m, nil
		}
	case "esc":
		m.Phase = PhaseMenu
	default:
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.runFuzzyFiltering()
	}
	return m, cmd
}

func (m WelcomeModel) updateProjectMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedProjectOption = navigateList(m.SelectedProjectOption, len(m.ProjectOptions), m.Settings.VimMode, msg)
	case "enter":
		return m.handleProjectMenuSelection()
	case "esc", "q":
		m.Phase = PhaseMenu
	}
	return m, nil
}

func (m WelcomeModel) updateProjectViewFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP, "k", "down", keyCtrlN, "j", "tab", keyShiftTab:
		m.SelectedProjectFile = navigateList(m.SelectedProjectFile, len(m.ProjectFiles), m.Settings.VimMode, msg)
	case "enter":
		if len(m.ProjectFiles) > 0 && m.SelectedProjectFile >= 0 && m.SelectedProjectFile < len(m.ProjectFiles) {
			fileName := m.ProjectFiles[m.SelectedProjectFile]
			filePath := filepath.Join(state.GetSessionDir(m.ProjectName), "output", fileName)
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
		}
	case "esc", "q":
		m.Phase = PhaseProjectMenu
	}
	return m, nil
}

func (m WelcomeModel) updateFileContentViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pageSize := 10
	if m.height > 12 {
		pageSize = m.height - 12
	}
	maxScroll := shared.MaxInt(0, len(m.ViewerLines)-pageSize)

	switch msg.String() {
	case "up", "k":
		m.ViewerScrollOffset = shared.MaxInt(0, m.ViewerScrollOffset-1)
	case "down", "j":
		m.ViewerScrollOffset = shared.MinInt(maxScroll, m.ViewerScrollOffset+1)
	case "pgup":
		m.ViewerScrollOffset = shared.MaxInt(0, m.ViewerScrollOffset-pageSize)
	case "pgdown", " ":
		m.ViewerScrollOffset = shared.MinInt(maxScroll, m.ViewerScrollOffset+pageSize)
	case "esc", "q":
		m.Phase = PhaseProjectViewFiles
	}
	return m, nil
}

func (m WelcomeModel) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "yes", "enter":
		if !m.IsNewProject {
			dir := state.GetSessionDir(m.ProjectName)
			_ = os.RemoveAll(dir)
		}
		m.alertTitle = "Project Deleted"
		m.alertMessage = fmt.Sprintf("Project '%s' and all its files have been deleted.", m.ProjectName)
		m.alertNext = PhaseMenu
		m.Phase = PhaseStatusAlert
	case "n", "no", "esc":
		m.Phase = PhaseProjectMenu
	}
	return m, nil
}

func (m *WelcomeModel) runFuzzyFiltering() {
	var filtered []string
	for _, proj := range m.Projects {
		if shared.FuzzyMatch(proj, m.filterInput.Value()) {
			filtered = append(filtered, proj)
		}
	}
	m.FilteredProjects = filtered

	if len(m.FilteredProjects) > 0 {
		m.SelectedProject = shared.MinInt(len(m.FilteredProjects)-1, shared.MaxInt(0, m.SelectedProject))
	} else {
		m.SelectedProject = 0
	}
}

func (m *WelcomeModel) adjustSettingFocus(delta int) {
	if m.SelectedSettingIdx < len(m.settingInputs) {
		m.settingInputs[m.SelectedSettingIdx].Blur()
	}

	totalSettings := len(m.settingInputs) + 2
	m.SelectedSettingIdx = (m.SelectedSettingIdx + delta + totalSettings) % totalSettings

	if m.SelectedSettingIdx < len(m.settingInputs) {
		m.settingInputs[m.SelectedSettingIdx].Focus()
	}
}

func (m *WelcomeModel) saveSettingsFromInputs() {
	var tSec, mRet int
	_, _ = fmt.Sscanf(m.settingInputs[0].Value(), "%d", &tSec)
	_, _ = fmt.Sscanf(m.settingInputs[1].Value(), "%d", &mRet)
	outFolder := strings.TrimSpace(m.settingInputs[2].Value())

	if tSec > 0 {
		m.Settings.TimeoutSeconds = tSec
	}
	if mRet >= 0 {
		m.Settings.MaxRetries = mRet
	}
	if outFolder != "" {
		m.Settings.DefaultOutputFolder = outFolder
	}

	_ = config.SaveSettings(m.Settings, true)
	_ = config.SaveSettings(m.Settings, false)

	logger.LogEvent("TUI", fmt.Sprintf("Saved settings: timeout_seconds=%d max_retries=%d default_output_folder=%s debug=%t vim_mode=%t", m.Settings.TimeoutSeconds, m.Settings.MaxRetries, m.Settings.DefaultOutputFolder, m.Settings.Debug, m.Settings.VimMode))
	_ = logger.Init(false, m.Settings.Debug)
	m.Phase = PhaseMenu
}

func (m WelcomeModel) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "up", keyShiftTab:
		m.adjustSettingFocus(-1)
	case "k":
		if m.Settings.VimMode {
			m.adjustSettingFocus(-1)
		}
	case "down", "tab":
		m.adjustSettingFocus(1)
	case "j":
		if m.Settings.VimMode {
			m.adjustSettingFocus(1)
		}
	case " ", "space":
		if m.SelectedSettingIdx == len(m.settingInputs) {
			m.Settings.Debug = !m.Settings.Debug
			logger.LogEvent("TUI", fmt.Sprintf("Debug logging toggled: %t", m.Settings.Debug))
		} else if m.SelectedSettingIdx == len(m.settingInputs)+1 {
			m.Settings.VimMode = !m.Settings.VimMode
			logger.LogEvent("TUI", fmt.Sprintf("Vim mode toggled: %t", m.Settings.VimMode))
		}
	case "enter":
		m.saveSettingsFromInputs()
	case "esc":
		m.Phase = PhaseMenu
	default:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx], cmd = m.settingInputs[m.SelectedSettingIdx].Update(msg)
		}
	}
	return m, cmd
}
