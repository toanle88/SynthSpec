package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
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
	keyCtrlP    = "ctrl+p"
	keyCtrlN    = "ctrl+n"
	keyShiftTab = "shift+tab"
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

	// Project Resume Selection
	Projects         []string
	FilteredProjects []string
	SelectedProject  int
	filterInput      textinput.Model

	// TextInput for name
	textInput textinput.Model

	// Blueprint Selection
	Blueprints        []config.Blueprint
	SelectedBlueprint string
	SelectedBPIdx     int

	// Alerts
	alertTitle   string
	alertMessage string

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
		Phase:          PhaseMenu,
		Action:         ActionNone,
		Options:        []string{"Create New Project", "Resume Existing Project", "Export to Static HTML", "View Assets", "Audit Workspace", "Settings", "Exit"},
		SelectedOption: 0,
		textInput:      ti,
		filterInput:    fi,
		Blueprints:     blueprints,
		Settings:       settings,
		settingInputs:  []textinput.Model{tInput, rInput, oInput},
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
	case PhaseStatusAlert:
		m.Phase = PhaseMenu
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
		m.Action = ActionCreate
		return m, tea.Quit
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
			m.Action = ActionResume
			return m, tea.Quit
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
			m.Action = ActionExport
			return m, tea.Quit
		}
	case "esc":
		m.Phase = PhaseMenu
	default:
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.runFuzzyFiltering()
	}
	return m, cmd
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
	}

	body := lipgloss.JoinVertical(lipgloss.Left, logoText, subTitle, content)
	h := 18
	if m.Phase == PhaseBlueprintSelect || m.Phase == PhaseSettings {
		h = 22
	}
	styledBody := MainPanelStyle.Width(65).Height(h).Render(body)
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
	lines = append(lines, "", lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Up/Down to navigate, Enter to Save settings, Esc to cancel"))
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
