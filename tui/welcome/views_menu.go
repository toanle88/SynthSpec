package welcome

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

func (m WelcomeModel) viewMenu() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("Welcome to SynthSpec! Select an action to begin:"), "")
	for i, opt := range m.Options {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(shared.ColorText)
		if i == m.SelectedOption {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
		}
		label := opt
		if opt == "View Assets" || opt == "Audit Workspace" {
			label = fmt.Sprintf("%s %s", opt, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("(Coming Soon)"))
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(label)))
	}
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewCreateInput() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("⚡ Create New Engineering Project"), "")
	lines = append(lines, fmt.Sprintf("Project Name: %s", m.textInput.View()), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to continue, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewBlueprintSelect() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("🌱 Choose a Starting Blueprint"), "")

	indicator := " "
	style := lipgloss.NewStyle().Foreground(shared.ColorText)
	if m.SelectedBPIdx == 0 {
		indicator = "➔"
		style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
	}
	lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render("None (Start from scratch)")))
	lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Start with an empty specification session.")), "")

	for i, bp := range m.Blueprints {
		bpIdx := i + 1
		indicator = " "
		style = lipgloss.NewStyle().Foreground(shared.ColorText)
		if bpIdx == m.SelectedBPIdx {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(" %s %s (%s)", indicator, style.Render(bp.Name), bp.ID))
		lines = append(lines, fmt.Sprintf("     %s", lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(bp.Description)), "")
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to select blueprint, Esc to go back"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewStatusAlert() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Foreground(shared.ColorInfo).Render(m.alertTitle), "")
	lines = append(lines, m.alertMessage, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("[ Press any key to return to menu ]"))
	return strings.Join(lines, "\n")
}
