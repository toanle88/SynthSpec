package welcome

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

func (m WelcomeModel) viewSettings() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("⚙️ Global & Workspace Settings"), "")
	settingFields := []string{"API Timeout (seconds)", "Max API Retries", "Default Output Folder", "Debug Logging (opt-in)", "Vim Keybindings (hjkl)"}
	for i, field := range settingFields {
		prefix := "  "
		labelStyle := lipgloss.NewStyle().Foreground(shared.ColorText)
		if i == m.SelectedSettingIdx {
			prefix = "➔ "
			labelStyle = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
		}
		var valView string
		if i < len(m.settingInputs) {
			valView = m.settingInputs[i].View()
		} else if i == len(m.settingInputs) {
			if m.Settings.Debug {
				valView = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true).Render("[x] Enabled (press Space to toggle)")
			} else {
				valView = lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("[ ] Disabled (press Space to toggle)")
			}
		} else {
			if m.Settings.VimMode {
				valView = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true).Render("[x] Enabled (press Space to toggle)")
			} else {
				valView = lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("[ ] Disabled (press Space to toggle)")
			}
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, labelStyle.Render(field), valView))
	}
	lines = append(lines, "",
		fmt.Sprintf("  %s    %s",
			lipgloss.NewStyle().Background(shared.ColorSuccess).Foreground(shared.ColorBg).Padding(0, 1).Bold(true).Render("[ Save Settings ]"),
			lipgloss.NewStyle().Background(shared.ColorBorder).Foreground(shared.ColorText).Padding(0, 1).Render(cancelLiteral),
		),
		"",
		lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Or use keyboard: Enter to Save, Esc to cancel"),
	)
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewViewAssets() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("📂 View Assets (Interactive Viewer)"), "")
	lines = append(lines, "The interactive visual spec asset list and markdown reader is")
	lines = append(lines, "currently scheduled for implementation under Milestone 6.", "")
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("For now, please inspect files in the generated output directories directly."), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Esc or q to return to the main menu."))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewAuditWorkspace() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("🔍 Audit Workspace (Drift Detection)"), "")
	lines = append(lines, "The Workspace Auditor scans local physical source code and compares it")
	lines = append(lines, "against the established spec to flag interface drift or security violations.")
	lines = append(lines, "This compliance enforcement engine is planned for later Milestone 11.", "")
	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Esc or q to return to the main menu."))
	return strings.Join(lines, "\n")
}
