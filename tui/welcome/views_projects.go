package welcome

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

func (m WelcomeModel) viewResumeSelect() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("📂 Select Project to Resume"), "")
	lines = append(lines, fmt.Sprintf("Search: %s", m.filterInput.View()), "")

	if len(m.FilteredProjects) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("No matching projects found."), "")
	} else {
		for i, proj := range m.FilteredProjects {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(shared.ColorText)
			if i == m.SelectedProject {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(proj)))
		}
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to resume project, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewExportSelect() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render("📂 Select Project to Export to HTML"), "")
	lines = append(lines, fmt.Sprintf("Search: %s", m.filterInput.View()), "")

	if len(m.FilteredProjects) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("No matching projects found."), "")
	} else {
		for i, proj := range m.FilteredProjects {
			indicator := " "
			style := lipgloss.NewStyle().Foreground(shared.ColorText)
			if i == m.SelectedProject {
				indicator = "➔"
				style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
			}
			lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(proj)))
		}
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to export project, Esc to return to menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewProjectMenu() string {
	var lines []string
	status := "(Existing)"
	if m.IsNewProject {
		status = "(New)"
	}
	lines = append(lines, "", shared.TitleStyle.Render(fmt.Sprintf("📂 Project: %s %s", m.ProjectName, status)), "")
	for i, opt := range m.ProjectOptions {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(shared.ColorText)
		if i == m.SelectedProjectOption {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(opt)))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to select, Esc to return to main menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewProjectViewFiles() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render(fmt.Sprintf("📂 Project Files: %s", m.ProjectName)), "")
	for i, fName := range m.ProjectFiles {
		indicator := " "
		style := lipgloss.NewStyle().Foreground(shared.ColorText)
		if i == m.SelectedProjectFile {
			indicator = "➔"
			style = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
		}
		lines = append(lines, fmt.Sprintf(selectionFormat, indicator, style.Render(fName)))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("Press Enter to view, Esc to return to project menu"))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewFileContentViewer() string {
	pageSize := 10
	if m.height > 12 {
		pageSize = m.height - 12
	}

	var lines []string
	lines = append(lines, "", shared.TitleStyle.Render(fmt.Sprintf("📄 %s", m.ProjectFiles[m.SelectedProjectFile])), "")

	start := m.ViewerScrollOffset
	end := start + pageSize
	if end > len(m.ViewerLines) {
		end = len(m.ViewerLines)
	}

	for _, line := range m.ViewerLines[start:end] {
		lines = append(lines, line)
	}

	lines = append(lines, "", lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(fmt.Sprintf("Page: line %d-%d of %d  |  Use ↑/↓ scroll, Esc/q to go back", start+1, end, len(m.ViewerLines))))
	return strings.Join(lines, "\n")
}

func (m WelcomeModel) viewDeleteConfirm() string {
	var lines []string
	lines = append(lines, "", shared.TitleStyle.Foreground(shared.ColorWarning).Render("⚠️  Confirm Project Deletion"), "")
	lines = append(lines, fmt.Sprintf("Are you sure you want to delete project '%s'?", m.ProjectName), "")
	lines = append(lines, "This action cannot be undone. All files and sessions will be permanently removed.", "")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s    %s",
		lipgloss.NewStyle().Background(lipgloss.Color("#ef4444")).Foreground(shared.ColorBg).Padding(0, 1).Bold(true).Render("[ Yes, Delete ]"),
		lipgloss.NewStyle().Background(shared.ColorBorder).Foreground(shared.ColorText).Padding(0, 1).Render(cancelLiteral),
	))
	return strings.Join(lines, "\n")
}
