package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
)

type WelcomePhase int

const (
	PhaseMenu WelcomePhase = iota
	PhaseCreateInput
	PhaseBlueprintSelect
	PhaseResumeSelect
	PhaseStatusAlert
	PhaseSettings
	PhaseViewAssets
	PhaseAuditWorkspace
	PhaseExportSelect
	PhaseProjectMenu
	PhaseProjectViewFiles
	PhaseFileContentViewer
	PhaseDeleteConfirm
)

type WelcomeAction int

const (
	ActionNone WelcomeAction = iota
	ActionCreate
	ActionResume
	ActionExport
	ActionExit
)

const (
	keyCtrlP       = "ctrl+p"
	keyCtrlN       = "ctrl+n"
	keyShiftTab    = "shift+tab"
	cancelLiteral  = "[ Cancel ]"
	noFilesLiteral = "No Files"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type WelcomeModel struct {
	Phase          WelcomePhase
	Action         WelcomeAction
	ProjectName    string
	SelectedOption int
	Options        []string

	// Project Resume/Menu Selection
	Projects              []string
	FilteredProjects      []string
	SelectedProject       int
	filterInput           textinput.Model
	IsNewProject          bool
	ProjectOptions        []string
	SelectedProjectOption int

	// Project File Viewer
	ProjectFiles        []string
	SelectedProjectFile int
	ViewerLines         []string
	ViewerScrollOffset  int

	// TextInput for name
	textInput textinput.Model

	// Blueprint Selection
	Blueprints        []config.Blueprint
	SelectedBlueprint string
	SelectedBPIdx     int

	// Alerts
	alertTitle   string
	alertMessage string
	alertNext    WelcomePhase // phase to go to after closing the alert

	// Global Settings Phase Fields
	Settings           *config.Settings
	SelectedSettingIdx int
	settingInputs      []textinput.Model

	// Terminal dimensions
	width  int
	height int
}

func NewWelcomeModel() WelcomeModel {
	ti := textinput.New()
	ti.Placeholder = "Enter project name..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 30

	fi := textinput.New()
	fi.Placeholder = "Type to search..."
	fi.CharLimit = 64
	fi.Width = 30

	blueprints, _ := config.LoadBlueprints()
	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = &config.Settings{
			TimeoutSeconds:      config.DefaultTimeoutSeconds,
			MaxRetries:          config.DefaultMaxRetries,
			DefaultOutputFolder: config.DefaultOutputFolderValue,
		}
	}

	tInput := textinput.New()
	tInput.Placeholder = "60"
	tInput.CharLimit = 5
	tInput.Width = 10

	rInput := textinput.New()
	rInput.Placeholder = "3"
	rInput.CharLimit = 5
	rInput.Width = 10

	oInput := textinput.New()
	oInput.Placeholder = "./output"
	oInput.CharLimit = 256
	oInput.Width = 30

	return WelcomeModel{
		Phase:                 PhaseMenu,
		Action:                ActionNone,
		Options:               []string{"Create New Project", "Resume Existing Project", "Export to Static HTML", "View Assets", "Audit Workspace", "Settings", "Exit"},
		SelectedOption:        0,
		ProjectOptions:        []string{"Start/Resume Specification", "View Generated Files", "Export to Static HTML", "Delete Project", "Back to Main Menu"},
		SelectedProjectOption: 0,
		textInput:             ti,
		filterInput:           fi,
		Blueprints:            blueprints,
		Settings:              settings,
		settingInputs:         []textinput.Model{tInput, rInput, oInput},
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return textinput.Blink
}

const selectionFormat = " %s %s"

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
		m.SelectedOption = maxInt(0, m.SelectedOption-1)
	case PhaseBlueprintSelect:
		m.SelectedBPIdx = maxInt(0, m.SelectedBPIdx-1)
	case PhaseResumeSelect:
		m.SelectedProject = maxInt(0, m.SelectedProject-1)
	case PhaseExportSelect:
		m.SelectedProject = maxInt(0, m.SelectedProject-1)
	case PhaseProjectMenu:
		m.SelectedProjectOption = maxInt(0, m.SelectedProjectOption-1)
	case PhaseProjectViewFiles:
		m.SelectedProjectFile = maxInt(0, m.SelectedProjectFile-1)
	case PhaseFileContentViewer:
		m.ViewerScrollOffset = maxInt(0, m.ViewerScrollOffset-1)
	case PhaseSettings:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Blur()
		}
		m.SelectedSettingIdx = maxInt(0, m.SelectedSettingIdx-1)
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Focus()
		}
	}
	return m
}

func (m WelcomeModel) handleMouseWheelDown() WelcomeModel {
	switch m.Phase {
	case PhaseMenu:
		m.SelectedOption = minInt(len(m.Options)-1, m.SelectedOption+1)
	case PhaseBlueprintSelect:
		m.SelectedBPIdx = minInt(len(m.Blueprints), m.SelectedBPIdx+1)
	case PhaseResumeSelect:
		m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case PhaseExportSelect:
		m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case PhaseProjectMenu:
		m.SelectedProjectOption = minInt(len(m.ProjectOptions)-1, m.SelectedProjectOption+1)
	case PhaseProjectViewFiles:
		m.SelectedProjectFile = minInt(len(m.ProjectFiles)-1, m.SelectedProjectFile+1)
	case PhaseFileContentViewer:
		pageSize := 10
		if m.height > 12 {
			pageSize = m.height - 12
		}
		maxScroll := maxInt(0, len(m.ViewerLines)-pageSize)
		m.ViewerScrollOffset = minInt(maxScroll, m.ViewerScrollOffset+1)
	case PhaseSettings:
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Blur()
		}
		m.SelectedSettingIdx = minInt(4, m.SelectedSettingIdx+1)
		if m.SelectedSettingIdx < len(m.settingInputs) {
			m.settingInputs[m.SelectedSettingIdx].Focus()
		}
	}
	return m
}

func (m WelcomeModel) handleMouseLeftClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	rendered := stripANSI(m.View())
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
			m.SelectedProjectOption = 2 // default to export
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
			highlighted := HighlightMarkdown(string(contentBytes))
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

func (m WelcomeModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP:
		m.SelectedOption = maxInt(0, m.SelectedOption-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedOption = maxInt(0, m.SelectedOption-1)
		}
	case "down", keyCtrlN:
		m.SelectedOption = minInt(len(m.Options)-1, m.SelectedOption+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedOption = minInt(len(m.Options)-1, m.SelectedOption+1)
		}
	case "tab":
		m.SelectedOption = (m.SelectedOption + 1) % len(m.Options)
	case keyShiftTab:
		m.SelectedOption = (m.SelectedOption - 1 + len(m.Options)) % len(m.Options)
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
	case "up", keyCtrlP:
		m.SelectedBPIdx = maxInt(0, m.SelectedBPIdx-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedBPIdx = maxInt(0, m.SelectedBPIdx-1)
		}
	case "down", keyCtrlN:
		m.SelectedBPIdx = minInt(len(m.Blueprints), m.SelectedBPIdx+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedBPIdx = minInt(len(m.Blueprints), m.SelectedBPIdx+1)
		}
	case "tab":
		m.SelectedBPIdx = (m.SelectedBPIdx + 1) % (len(m.Blueprints) + 1)
	case keyShiftTab:
		m.SelectedBPIdx = (m.SelectedBPIdx - 1 + len(m.Blueprints) + 1) % (len(m.Blueprints) + 1)
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
	case "up", keyCtrlP:
		m.SelectedProject = maxInt(0, m.SelectedProject-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedProject = maxInt(0, m.SelectedProject-1)
		}
	case "down", keyCtrlN:
		m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
		}
	case "tab":
		if len(m.FilteredProjects) > 0 {
			m.SelectedProject = (m.SelectedProject + 1) % len(m.FilteredProjects)
		}
	case keyShiftTab:
		if len(m.FilteredProjects) > 0 {
			m.SelectedProject = (m.SelectedProject - 1 + len(m.FilteredProjects)) % len(m.FilteredProjects)
		}
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
	case "up", keyCtrlP:
		m.SelectedProject = maxInt(0, m.SelectedProject-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedProject = maxInt(0, m.SelectedProject-1)
		}
	case "down", keyCtrlN:
		m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedProject = minInt(len(m.FilteredProjects)-1, m.SelectedProject+1)
		}
	case "tab":
		if len(m.FilteredProjects) > 0 {
			m.SelectedProject = (m.SelectedProject + 1) % len(m.FilteredProjects)
		}
	case keyShiftTab:
		if len(m.FilteredProjects) > 0 {
			m.SelectedProject = (m.SelectedProject - 1 + len(m.FilteredProjects)) % len(m.FilteredProjects)
		}
	case "enter":
		if len(m.FilteredProjects) > 0 && m.SelectedProject >= 0 && m.SelectedProject < len(m.FilteredProjects) {
			m.ProjectName = m.FilteredProjects[m.SelectedProject]
			m.IsNewProject = false
			m.SelectedProjectOption = 2 // default to export
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
	case "up", keyCtrlP:
		m.SelectedProjectOption = maxInt(0, m.SelectedProjectOption-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedProjectOption = maxInt(0, m.SelectedProjectOption-1)
		}
	case "down", keyCtrlN:
		m.SelectedProjectOption = minInt(len(m.ProjectOptions)-1, m.SelectedProjectOption+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedProjectOption = minInt(len(m.ProjectOptions)-1, m.SelectedProjectOption+1)
		}
	case "tab":
		m.SelectedProjectOption = (m.SelectedProjectOption + 1) % len(m.ProjectOptions)
	case keyShiftTab:
		m.SelectedProjectOption = (m.SelectedProjectOption - 1 + len(m.ProjectOptions)) % len(m.ProjectOptions)
	case "enter":
		return m.handleProjectMenuSelection()
	case "esc", "q":
		m.Phase = PhaseMenu
	}
	return m, nil
}

func (m WelcomeModel) handleProjectMenuSelection() (tea.Model, tea.Cmd) {
	switch m.SelectedProjectOption {
	case 0: // Start/Resume Specification
		if m.IsNewProject {
			m.Action = ActionCreate
		} else {
			m.Action = ActionResume
		}
		return m, tea.Quit

	case 1: // View Generated Files
		return m.handleProjectMenuSelectionViewFiles()

	case 2: // Export to Static HTML
		return m.handleProjectMenuSelectionExport()

	case 3: // Delete Project
		m.Phase = PhaseDeleteConfirm
		return m, nil

	case 4: // Back to Main Menu
		m.Phase = PhaseMenu
		return m, nil
	}
	return m, nil
}

func (m WelcomeModel) handleProjectMenuSelectionViewFiles() (tea.Model, tea.Cmd) {
	if m.IsNewProject {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "This is a new project. No specifications have been generated yet."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	outDir := filepath.Join(state.GetSessionDir(m.ProjectName), "output")
	files, err := os.ReadDir(outDir)
	if err != nil || len(files) == 0 {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "No generated markdown specifications were found for this project."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}

	var mdFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
			mdFiles = append(mdFiles, f.Name())
		}
	}
	if len(mdFiles) == 0 {
		m.alertTitle = noFilesLiteral
		m.alertMessage = "No generated markdown specifications were found for this project."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	m.ProjectFiles = mdFiles
	m.SelectedProjectFile = 0
	m.Phase = PhaseProjectViewFiles
	return m, nil
}

func (m WelcomeModel) handleProjectMenuSelectionExport() (tea.Model, tea.Cmd) {
	if m.IsNewProject {
		m.alertTitle = "No Specifications"
		m.alertMessage = "This is a new project. Start specification generation before exporting."
		m.alertNext = PhaseProjectMenu
		m.Phase = PhaseStatusAlert
		return m, nil
	}
	projDir := state.GetSessionDir(m.ProjectName)
	outputDir := filepath.Join(projDir, "output")
	distDir := filepath.Join(projDir, "dist")

	indexPath, err := generator.ExportToHTML(m.ProjectName, outputDir, distDir)
	if err != nil {
		m.alertTitle = "Export Failed"
		m.alertMessage = fmt.Sprintf("Failed to export: %v", err)
	} else {
		m.alertTitle = "Export Successful"
		m.alertMessage = fmt.Sprintf("HTML exported successfully to:\n%s", indexPath)
	}
	m.alertNext = PhaseProjectMenu
	m.Phase = PhaseStatusAlert
	return m, nil
}

func (m WelcomeModel) updateProjectViewFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", keyCtrlP:
		m.SelectedProjectFile = maxInt(0, m.SelectedProjectFile-1)
	case "k":
		if m.Settings.VimMode {
			m.SelectedProjectFile = maxInt(0, m.SelectedProjectFile-1)
		}
	case "down", keyCtrlN:
		m.SelectedProjectFile = minInt(len(m.ProjectFiles)-1, m.SelectedProjectFile+1)
	case "j":
		if m.Settings.VimMode {
			m.SelectedProjectFile = minInt(len(m.ProjectFiles)-1, m.SelectedProjectFile+1)
		}
	case "tab":
		m.SelectedProjectFile = (m.SelectedProjectFile + 1) % len(m.ProjectFiles)
	case keyShiftTab:
		m.SelectedProjectFile = (m.SelectedProjectFile - 1 + len(m.ProjectFiles)) % len(m.ProjectFiles)
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
			highlighted := HighlightMarkdown(string(contentBytes))
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
	maxScroll := maxInt(0, len(m.ViewerLines)-pageSize)

	switch msg.String() {
	case "up", "k":
		m.ViewerScrollOffset = maxInt(0, m.ViewerScrollOffset-1)
	case "down", "j":
		m.ViewerScrollOffset = minInt(maxScroll, m.ViewerScrollOffset+1)
	case "pgup":
		m.ViewerScrollOffset = maxInt(0, m.ViewerScrollOffset-pageSize)
	case "pgdown", " ":
		m.ViewerScrollOffset = minInt(maxScroll, m.ViewerScrollOffset+pageSize)
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
		if fuzzyMatch(proj, m.filterInput.Value()) {
			filtered = append(filtered, proj)
		}
	}
	m.FilteredProjects = filtered

	if len(m.FilteredProjects) > 0 {
		m.SelectedProject = minInt(len(m.FilteredProjects)-1, maxInt(0, m.SelectedProject))
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

func (m *WelcomeModel) handleMenuSelection() {
	switch m.SelectedOption {
	case 0:
		m.textInput.SetValue("")
		m.Phase = PhaseCreateInput
	case 1:
		projects, err := state.ListProjects()
		if err != nil {
			m.alertTitle = "Error Scanning Projects"
			m.alertMessage = fmt.Sprintf("Failed to list existing projects: %v", err)
			m.Phase = PhaseStatusAlert
			return
		}
		if len(projects) == 0 {
			m.alertTitle = "No Saved Projects"
			m.alertMessage = "No active SynthSpec projects were found in this directory.\nChoose 'Create New Project' to get started."
			m.Phase = PhaseStatusAlert
			return
		}
		m.Projects = projects
		m.FilteredProjects = projects
		m.SelectedProject = 0
		m.filterInput.SetValue("")
		m.filterInput.Focus()
		m.Phase = PhaseResumeSelect
	case 2:
		projects, err := state.ListProjects()
		if err != nil {
			m.alertTitle = "Error Scanning Projects"
			m.alertMessage = fmt.Sprintf("Failed to list existing projects: %v", err)
			m.Phase = PhaseStatusAlert
			return
		}
		if len(projects) == 0 {
			m.alertTitle = "No Saved Projects"
			m.alertMessage = "No active SynthSpec projects were found to export.\nChoose 'Create New Project' to get started."
			m.Phase = PhaseStatusAlert
			return
		}
		m.Projects = projects
		m.FilteredProjects = projects
		m.SelectedProject = 0
		m.filterInput.SetValue("")
		m.filterInput.Focus()
		m.Phase = PhaseExportSelect
	case 3:
		m.Phase = PhaseViewAssets
	case 4:
		m.Phase = PhaseThemeToggle() // Or keep PhaseAuditWorkspace as planned
		m.Phase = PhaseAuditWorkspace
	case 5:
		m.settingInputs[0].SetValue(fmt.Sprintf("%d", m.Settings.TimeoutSeconds))
		m.settingInputs[1].SetValue(fmt.Sprintf("%d", m.Settings.MaxRetries))
		m.settingInputs[2].SetValue(m.Settings.DefaultOutputFolder)
		m.SelectedSettingIdx = 0
		m.settingInputs[0].Focus()
		m.settingInputs[1].Blur()
		m.settingInputs[2].Blur()
		m.Phase = PhaseSettings
	case 6:
		m.Action = ActionExit
		m.Phase = PhaseStatusAlert
	}
}

func PhaseThemeToggle() WelcomePhase {
	return PhaseAuditWorkspace
}

func (m WelcomeModel) View() string {
	if m.Action == ActionExit && m.Phase == PhaseStatusAlert {
		return "Exiting SynthSpec. Goodbye!\n"
	}

	logo := `
   _____             __  __   _____                     
  / ____|           |  \/  | / ____|                    
 | (___   _   _  _  | \  / || (___   _ __    ___   ___  
  \___ \ | | | || |_| |\/| | \___ \ | '_ \  / _ \ / __| 
  ____) || |_| ||  _| |  | | ____) || |_) ||  __/| (__  
 |_____/  \__, ||_| |_|  |_||_____/ | .__/  \___| \___| 
           __/ |                    | |                 
          |___/                     |_|                 
`
	logoStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	logoText := logoStyle.Render(logo)
	subTitle := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("  Open-Source BYOK AI Solution Architect CLI")

	var content string
	switch m.Phase {
	case PhaseMenu:
		content = m.viewMenu()
	case PhaseCreateInput:
		content = m.viewCreateInput()
	case PhaseBlueprintSelect:
		content = m.viewBlueprintSelect()
	case PhaseResumeSelect:
		content = m.viewResumeSelect()
	case PhaseExportSelect:
		content = m.viewExportSelect()
	case PhaseStatusAlert:
		content = m.viewStatusAlert()
	case PhaseSettings:
		content = m.viewSettings()
	case PhaseViewAssets:
		content = m.viewViewAssets()
	case PhaseAuditWorkspace:
		content = m.viewAuditWorkspace()
	case PhaseProjectMenu:
		content = m.viewProjectMenu()
	case PhaseProjectViewFiles:
		content = m.viewProjectViewFiles()
	case PhaseFileContentViewer:
		content = m.viewFileContentViewer()
	case PhaseDeleteConfirm:
		content = m.viewDeleteConfirm()
	}

	body := lipgloss.JoinVertical(lipgloss.Left, logoText, subTitle, content)
	h := 18
	w := 65
	switch m.Phase {
	case PhaseBlueprintSelect, PhaseSettings:
		h = 22
	case PhaseFileContentViewer:
		if m.height > 6 {
			h = m.height - 4
		}
		if m.width > 8 {
			w = m.width - 6
		}
	}
	styledBody := MainPanelStyle.Width(w).Height(h).Render(body)
	return DocStyle.Render(styledBody)
}

func (m WelcomeModel) viewMenu() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("Welcome to SynthSpec! Select an action to begin:"), "")
	for i, opt := range m.Options {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.SelectedOption {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		label := opt
		if opt == "View Assets" || opt == "Audit Workspace" {
			label = fmt.Sprintf("%s %s", opt, lipgloss.NewStyle().Foreground(ColorMuted).Render("(Coming Soon)"))
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(label)))
	}
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewCreateInput() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("⚡ Create New Engineering Project"), "")
	lines = append(lines, fmt.Sprintf("Project Name: %s", m.textInput.View()), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to continue, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewBlueprintSelect() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("🌱 Choose a Starting Blueprint"), "")

	indicator := " "
	style := lipgloss.NewStyle().Foreground(ColorText)
	if m.SelectedBPIdx == 0 {
		indicator = "➔"
		style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	}
	lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render("None (Start from scratch)")))
	lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(ColorMuted).Render("Start with an empty specification session.")), "")

	for i, bp := range m.Blueprints {
		bpIdx := i + 1
		indicator = " "
		style = lipgloss.NewStyle().Foreground(ColorText)
		if bpIdx == m.SelectedBPIdx {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(" %s %s (%s)", indicator, style.Render(bp.Name), bp.ID))
		lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(ColorMuted).Render(bp.Description)), "")
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to select blueprint, Esc to go back"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewResumeSelect() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("📂 Select Project to Resume"), "")
	lines = append(lines, fmt.Sprintf("Search: %s", m.filterInput.View()), "")
	
	if len(m.FilteredProjects) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(ColorMuted).Render("No matching projects found."), "")
	} else {
		for i, proj := range m.FilteredProjects {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(ColorText)
			if i == m.SelectedProject {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(proj)))
		}
		lines = append(lines, "")
	}
	
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to resume project, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewExportSelect() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("📂 Select Project to Export to HTML"), "")
	lines = append(lines, fmt.Sprintf("Search: %s", m.filterInput.View()), "")
	
	if len(m.FilteredProjects) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(ColorMuted).Render("No matching projects found."), "")
	} else {
		for i, proj := range m.FilteredProjects {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(ColorText)
			if i == m.SelectedProject {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(proj)))
		}
		lines = append(lines, "")
	}
	
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to export project, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewStatusAlert() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Foreground(ColorInfo).Render(m.alertTitle), "")
	lines = append(lines, m.alertMessage, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("[ Press any key to return to menu ]"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewSettings() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("⚙️ Global & Workspace Settings"), "")
	settingFields := []string{"API Timeout (seconds)", "Max API Retries", "Default Output Folder", "Debug Logging (opt-in)", "Vim Keybindings (hjkl)"}
	for i, field := range settingFields {
		prefix := "  "
		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.SelectedSettingIdx {
			prefix = "➔ "
			labelStyle = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		var valView string
		if i < len(m.settingInputs) {
			valView = m.settingInputs[i].View()
		} else if i == len(m.settingInputs) {
			if m.Settings.Debug {
				valView = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("[x] Enabled (press Space to toggle)")
			} else {
				valView = lipgloss.NewStyle().Foreground(ColorMuted).Render("[ ] Disabled (press Space to toggle)")
			}
		} else {
			if m.Settings.VimMode {
				valView = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("[x] Enabled (press Space to toggle)")
			} else {
				valView = lipgloss.NewStyle().Foreground(ColorMuted).Render("[ ] Disabled (press Space to toggle)")
			}
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, labelStyle.Render(field), valView))
	}
	lines = append(lines, "",
		fmt.Sprintf("  %s    %s",
			lipgloss.NewStyle().Background(ColorSuccess).Foreground(ColorBg).Padding(0, 1).Bold(true).Render("[ Save Settings ]"),
			lipgloss.NewStyle().Background(ColorBorder).Foreground(ColorText).Padding(0, 1).Render(cancelLiteral),
		),
		"",
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Or use keyboard: Enter to Save, Esc to cancel"),
	)
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewViewAssets() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("📂 View Assets (Interactive Viewer)"), "")
	lines = append(lines, "The interactive visual spec asset list and markdown reader is")
	lines = append(lines, "currently scheduled for implementation under Milestone 6.", "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("For now, please inspect files in the generated output directories directly."), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Esc or q to return to the main menu."))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewAuditWorkspace() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render("🔍 Audit Workspace (Drift Detection)"), "")
	lines = append(lines, "The Workspace Auditor scans local physical source code and compares it")
	lines = append(lines, "against the established spec to flag interface drift or security violations.")
	lines = append(lines, "This compliance enforcement engine is planned for later Milestone 11.", "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Esc or q to return to the main menu."))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewProjectMenu() string {
	var lines []string
	status := "(Existing)"
	if m.IsNewProject {
		status = "(New)"
	}
	lines = append(lines, "", TitleStyle.Render(fmt.Sprintf("📂 Project: %s %s", m.ProjectName, status)), "")
	for i, opt := range m.ProjectOptions {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.SelectedProjectOption {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(opt)))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to select, Esc to return to main menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewProjectViewFiles() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Render(fmt.Sprintf("📂 Project Files: %s", m.ProjectName)), "")
	for i, fName := range m.ProjectFiles {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.SelectedProjectFile {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(fName)))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to view, Esc to return to project menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewFileContentViewer() string {
	pageSize := 10
	if m.height > 12 {
		pageSize = m.height - 12
	}

	var lines []string
	fileName := ""
	if m.SelectedProjectFile >= 0 && m.SelectedProjectFile < len(m.ProjectFiles) {
		fileName = m.ProjectFiles[m.SelectedProjectFile]
	}
	lines = append(lines, TitleStyle.Render(fmt.Sprintf("📄 Viewing: %s", fileName)), "")

	start := m.ViewerScrollOffset
	end := start + pageSize
	if end > len(m.ViewerLines) {
		end = len(m.ViewerLines)
	}

	for i := start; i < end; i++ {
		lines = append(lines, m.ViewerLines[i])
	}

	lines = append(lines, "")
	pct := 0
	if len(m.ViewerLines) > 0 {
		pct = (end * 100) / len(m.ViewerLines)
	}
	scrollBar := RenderProgressBar(30, pct)
	lines = append(lines, fmt.Sprintf("Scroll: %s | Press Esc to exit viewer", scrollBar))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewDeleteConfirm() string {
	var lines []string
	lines = append(lines, "", TitleStyle.Foreground(ColorWarning).Render(fmt.Sprintf("⚠️ Delete Project: %s", m.ProjectName)), "")
	lines = append(lines, "Are you sure you want to permanently delete this project", "and all of its generated files? This action cannot be undone.", "")
	lines = append(lines, 
		fmt.Sprintf("  %s    %s",
			lipgloss.NewStyle().Background(lipgloss.Color("#ef4444")).Foreground(ColorBg).Padding(0, 1).Bold(true).Render("[ Yes, Delete ]"),
			lipgloss.NewStyle().Background(ColorBorder).Foreground(ColorText).Padding(0, 1).Render(cancelLiteral),
		),
		"",
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Or use keyboard: Y to Delete, N/Esc to cancel"),
	)
	return strings.Join(lines, "\n")
}

func fuzzyMatch(s, query string) bool {
	s = strings.ToLower(s)
	query = strings.ToLower(query)
	if query == "" {
		return true
	}
	sRunes := []rune(s)
	qRunes := []rune(query)
	sIdx := 0
	for _, qRune := range qRunes {
		found := false
		for sIdx < len(sRunes) {
			if sRunes[sIdx] == qRune {
				found = true
				sIdx++
				break
			}
			sIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
