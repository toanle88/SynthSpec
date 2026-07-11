package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

// getFileStatusIconAndStyle mapping maps dynamic status to its TUI icon and color style.
func (m DashboardModel) getFileStatusIconAndStyle(status string) (string, lipgloss.Style) {
	switch status {
	case "skipped", "done", "extracting":
		return "🟢 Done", shared.StyleSuccess
	case "waiting_approval":
		return "⏸️ Awaiting Approval", shared.StyleWarning
	case "synthesizing":
		return "🔄 Synthesizing", shared.StyleInfo
	case "correcting":
		return "⚠️ Correcting", shared.StyleWarning
	case "auditing":
		return "🔍 Auditing", shared.StyleInfo
	case "refining":
		return "🛠️ Refining", shared.StyleWarning
	case "failed":
		return "🔴 Failed", shared.StyleError
	default:
		return pendingStr, shared.StyleMuted
	}
}

// renderFileProgressList draws the complete layout list of generated files with indicators.
func (m DashboardModel) renderFileProgressList() string {
	var sourceLines []string
	var downstreamLines []string

	for idx, file := range m.genFiles {
		status := m.genFileStatuses[file]
		details := m.genFileDetails[file]

		icon, style := m.getFileStatusIconAndStyle(status)
		styledIcon := style.Bold(true).Render(icon)

		var styledFile string
		prefix := "  "
		if m.isCompleted && idx == m.selectedFileIdx {
			prefix = "❯ "
			styledFile = lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true).Render(file)
		} else {
			styledFile = lipgloss.NewStyle().Foreground(shared.ColorText).Bold(true).Render(file)
		}

		var line string
		if details != "" && details != "completed successfully" && details != "already generated" {
			styledDetails := lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(fmt.Sprintf("(%s)", details))
			line = fmt.Sprintf("%s%s %s %s", prefix, styledIcon, styledFile, styledDetails)
		} else {
			line = fmt.Sprintf("%s%s %s", prefix, styledIcon, styledFile)
		}

		if file == "01_domain_model_use_cases.md" {
			sourceLines = append(sourceLines, line)
		} else {
			downstreamLines = append(downstreamLines, line)
		}
	}

	var sections []string
	if len(sourceLines) > 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("Source"))
		sections = append(sections, sourceLines...)
	}
	if len(downstreamLines) > 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("Parallel downstream"))
		sections = append(sections, renderParallelProgressGrid(downstreamLines))
	}

	return strings.Join(sections, "\n")
}

func renderParallelProgressGrid(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	half := (len(lines) + 1) / 2
	leftLines := lines[:half]
	rightLines := lines[half:]

	leftBlock := strings.Join(leftLines, "\n")
	rightBlock := strings.Join(rightLines, "\n")

	if rightBlock == "" {
		return leftBlock
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftBlock,
		"    ",
		rightBlock,
	)
}
