package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/state"
)

type WelcomePhase int

const (
	PhaseMenu WelcomePhase = iota
	PhaseCreateInput
	PhaseResumeSelect
	PhaseStatusAlert
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

	// Alerts
	alertTitle   string
	alertMessage string

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

	return WelcomeModel{
		Phase:          PhaseMenu,
		Action:         ActionNone,
		Options:        []string{"Create New Project", "Resume Existing Project", "View Assets", "Audit Workspace", "Settings", "Exit"},
		SelectedOption: 0,
		textInput:      ti,
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
					m.Action = ActionCreate
					return m, tea.Quit
				}
			case tea.KeyEsc:
				m.Phase = PhaseMenu
			default:
				m.textInput, cmd = m.textInput.Update(msg)
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
		m.alertTitle = "View Assets"
		m.alertMessage = "Interactive Asset Viewer is planned for Milestone 6.\nFor now, please inspect files in the generated output directories directly."
		m.Phase = PhaseStatusAlert
	case 3: // Audit Workspace
		m.alertTitle = "Audit Workspace"
		m.alertMessage = "Compliance Audit and Drift Detection is planned for Milestone 5.\nThis will scan and check local code against synthesized specs."
		m.Phase = PhaseStatusAlert
	case 4: // Settings
		m.alertTitle = "Global Settings"
		m.alertMessage = "Settings Configuration Pane is planned for Milestone 5.\nThis will allow tweaking timeouts, retry thresholds, and export targets."
		m.Phase = PhaseStatusAlert
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
			if opt == "View Assets" || opt == "Audit Workspace" || opt == "Settings" {
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
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter to initialize, Esc to return to menu"))
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
	}

	// Main frame
	body := lipgloss.JoinVertical(lipgloss.Left, logoText, subTitle, content)
	styledBody := MainPanelStyle.
		Width(65).
		Height(18).
		Render(body)

	return DocStyle.Render(styledBody)
}
