package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
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
)

type WelcomeAction int

const (
	ActionNone WelcomeAction = iota
	ActionCreate
	ActionResume
	ActionExit
)

type WelcomeModel struct {
	Phase          WelcomePhase
	Action         WelcomeAction
	ProjectName    string
	SelectedOption int
	Options        []string

	// Project Resume Selection
	Projects        []string
	SelectedProject int

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
		Options:        []string{"Create New Project", "Resume Existing Project", "View Assets", "Audit Workspace", "Settings", "Exit"},
		SelectedOption: 0,
		textInput:      ti,
		Blueprints:     blueprints,
		Settings:       settings,
		settingInputs:  []textinput.Model{tInput, rInput, oInput},
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.Action = ActionExit
			return m, tea.Quit
		}

		switch m.Phase {
		case PhaseMenu:
			switch msg.String() {
			case "up", "k":
				if m.SelectedOption > 0 {
					m.SelectedOption--
				}
			case "down", "j":
				if m.SelectedOption < len(m.Options)-1 {
					m.SelectedOption++
				}
			case "enter":
				m.handleMenuSelection()
			case "q", "esc":
				m.Action = ActionExit
				return m, tea.Quit
			}

		case PhaseCreateInput:
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

		case PhaseBlueprintSelect:
			switch msg.String() {
			case "up", "k":
				if m.SelectedBPIdx > 0 {
					m.SelectedBPIdx--
				}
			case "down", "j":
				if m.SelectedBPIdx < len(m.Blueprints) {
					m.SelectedBPIdx++
				}
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

		case PhaseResumeSelect:
			switch msg.String() {
			case "up", "k":
				if m.SelectedProject > 0 {
					m.SelectedProject--
				}
			case "down", "j":
				if m.SelectedProject < len(m.Projects)-1 {
					m.SelectedProject++
				}
			case "enter":
				if len(m.Projects) > 0 && m.SelectedProject >= 0 && m.SelectedProject < len(m.Projects) {
					m.ProjectName = m.Projects[m.SelectedProject]
					m.Action = ActionResume
					return m, tea.Quit
				}
			case "esc", "q":
				m.Phase = PhaseMenu
			}

		case PhaseStatusAlert:
			// Press any key to go back
			m.Phase = PhaseMenu

		case PhaseSettings:
			switch msg.String() {
			case "up", "k", "shift+tab":
				m.settingInputs[m.SelectedSettingIdx].Blur()
				if m.SelectedSettingIdx > 0 {
					m.SelectedSettingIdx--
				} else {
					m.SelectedSettingIdx = len(m.settingInputs) - 1
				}
				m.settingInputs[m.SelectedSettingIdx].Focus()
			case "down", "j", "tab":
				m.settingInputs[m.SelectedSettingIdx].Blur()
				if m.SelectedSettingIdx < len(m.settingInputs)-1 {
					m.SelectedSettingIdx++
				} else {
					m.SelectedSettingIdx = 0
				}
				m.settingInputs[m.SelectedSettingIdx].Focus()
			case "enter":
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

				m.Phase = PhaseMenu
			case "esc":
				m.Phase = PhaseMenu
			default:
				m.settingInputs[m.SelectedSettingIdx], cmd = m.settingInputs[m.SelectedSettingIdx].Update(msg)
			}

		case PhaseViewAssets, PhaseAuditWorkspace:
			switch msg.String() {
			case "esc", "q", "enter":
				m.Phase = PhaseMenu
			}
		}
	}

	return m, cmd
}

func (m *WelcomeModel) handleMenuSelection() {
	switch m.SelectedOption {
	case 0: // Create New Project
		m.textInput.SetValue("")
		m.Phase = PhaseCreateInput
	case 1: // Resume Existing Project
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
		m.SelectedProject = 0
		m.Phase = PhaseResumeSelect
	case 2: // View Assets
		m.Phase = PhaseViewAssets
	case 3: // Audit Workspace
		m.Phase = PhaseAuditWorkspace
	case 4: // Settings
		m.settingInputs[0].SetValue(fmt.Sprintf("%d", m.Settings.TimeoutSeconds))
		m.settingInputs[1].SetValue(fmt.Sprintf("%d", m.Settings.MaxRetries))
		m.settingInputs[2].SetValue(m.Settings.DefaultOutputFolder)
		m.SelectedSettingIdx = 0
		m.settingInputs[0].Focus()
		m.settingInputs[1].Blur()
		m.settingInputs[2].Blur()
		m.Phase = PhaseSettings
	case 5: // Exit
		m.Action = ActionExit
		m.Phase = PhaseStatusAlert // or we could quit immediately
	}
}

func (m WelcomeModel) View() string {
	if m.Action == ActionExit && m.Phase == PhaseStatusAlert {
		return "Exiting SynthSpec. Goodbye!\n"
	}

	var content string

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

	switch m.Phase {
	case PhaseMenu:
		var menuLines []string
		menuLines = append(menuLines, "")
		menuLines = append(menuLines, TitleStyle.Render("Welcome to SynthSpec! Select an action to begin:"))
		menuLines = append(menuLines, "")

		for i, opt := range m.Options {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(ColorText)
			if i == m.SelectedOption {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}

			// Add visual indicator for coming soon
			label := opt
			if opt == "View Assets" || opt == "Audit Workspace" {
				label = fmt.Sprintf("%s %s", opt, lipgloss.NewStyle().Foreground(ColorMuted).Render("(Coming Soon)"))
			}

			menuLines = append(menuLines, fmt.Sprintf(" %s %s", indicator, style.Render(label)))
		}

		content = strings.Join(menuLines, "\n")

	case PhaseCreateInput:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("⚡ Create New Engineering Project"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Project Name: %s", m.textInput.View()))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to continue, Esc to return to menu"))
		content = strings.Join(lines, "\n")

	case PhaseBlueprintSelect:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("🌱 Choose a Starting Blueprint"))
		lines = append(lines, "")

		// None Option
		indicator := " "
		style := lipgloss.NewStyle().Foreground(ColorText)
		if m.SelectedBPIdx == 0 {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(" %s %s", indicator, style.Render("None (Start from scratch)")))
		lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(ColorMuted).Render("Start with an empty specification session.")))
		lines = append(lines, "")

		for i, bp := range m.Blueprints {
			bpIdx := i + 1
			indicator = " "
			style = lipgloss.NewStyle().Foreground(ColorText)
			if bpIdx == m.SelectedBPIdx {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(" %s %s (%s)", indicator, style.Render(bp.Name), bp.ID))
			lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(ColorMuted).Render(bp.Description)))
			lines = append(lines, "")
		}

		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to select blueprint, Esc to go back"))
		content = strings.Join(lines, "\n")

	case PhaseResumeSelect:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("📂 Select Project to Resume"))
		lines = append(lines, "")

		for i, proj := range m.Projects {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(ColorText)
			if i == m.SelectedProject {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(" %s %s", indicator, style.Render(proj)))
		}

		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to resume project, Esc to return to menu"))
		content = strings.Join(lines, "\n")

	case PhaseStatusAlert:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Foreground(ColorInfo).Render(m.alertTitle))
		lines = append(lines, "")
		lines = append(lines, m.alertMessage)
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("[ Press any key to return to menu ]"))
		content = strings.Join(lines, "\n")

	case PhaseSettings:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("⚙️ Global & Workspace Settings"))
		lines = append(lines, "")

		settingFields := []string{"API Timeout (seconds)", "Max API Retries", "Default Output Folder"}
		for i, field := range settingFields {
			prefix := "  "
			labelStyle := lipgloss.NewStyle().Foreground(ColorText)
			if i == m.SelectedSettingIdx {
				prefix = "➔ "
				labelStyle = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, labelStyle.Render(field), m.settingInputs[i].View()))
		}

		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Up/Down to navigate, Enter to Save settings, Esc to cancel"))
		content = strings.Join(lines, "\n")

	case PhaseViewAssets:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("📂 View Assets (Interactive Viewer)"))
		lines = append(lines, "")
		lines = append(lines, "The interactive visual spec asset list and markdown reader is")
		lines = append(lines, "currently scheduled for implementation under Milestone 6.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("For now, please inspect files in the generated output directories directly."))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Esc or q to return to the main menu."))
		content = strings.Join(lines, "\n")

	case PhaseAuditWorkspace:
		var lines []string
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("🔍 Audit Workspace (Drift Detection)"))
		lines = append(lines, "")
		lines = append(lines, "The Workspace Auditor scans local physical source code and compares it")
		lines = append(lines, "against the established spec to flag interface drift or security violations.")
		lines = append(lines, "This compliance enforcement engine is planned for later Milestone 11.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Esc or q to return to the main menu."))
		content = strings.Join(lines, "\n")
	}

	// Main frame
	body := lipgloss.JoinVertical(lipgloss.Left, logoText, subTitle, content)
	
	h := 18
	if m.Phase == PhaseBlueprintSelect || m.Phase == PhaseSettings {
		h = 22
	}
	styledBody := MainPanelStyle.
		Width(65).
		Height(h).
		Render(body)

	return DocStyle.Render(styledBody)
}
