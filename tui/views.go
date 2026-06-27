package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m DashboardModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing SynthSpec Dashboard..."
	}

	header := m.renderHeader()
	sidebar := m.renderSidebar()
	mainChat := m.renderMainChat()
	footer := m.renderFooter()

	// Compute widths
	sidebarWidth := 34
	chatWidth := m.width - sidebarWidth - 8
	if chatWidth < 40 {
		chatWidth = 40
	}

	// Height limits
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 4
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	// Apply styles and dimensions
	styledSidebar := SidebarStyle.
		Width(sidebarWidth).
		Height(bodyHeight).
		Render(sidebar)

	styledChat := MainPanelStyle.
		Width(chatWidth).
		Height(bodyHeight).
		Render(mainChat)

	// Combine body horizontally
	body := lipgloss.JoinHorizontal(lipgloss.Top, styledSidebar, styledChat)

	// Combine everything vertically
	fullView := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	return DocStyle.Render(fullView)
}

func (m DashboardModel) renderHeader() string {
	title := " 🛠️  SynthSpec Solution Architect Dashboard "
	
	// Average Score Calculation
	avgScore := (m.Session.Scores.Functional +
		m.Session.Scores.Structural +
		m.Session.Scores.Security +
		m.Session.Scores.Compliance) / 4

	meta := fmt.Sprintf("Project: %s | Provider: %s | Model: %s",
		lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render(m.Session.ProjectName),
		strings.ToUpper(m.Session.Provider),
		m.Session.Model,
	)

	progBar := RenderProgressBar(40, avgScore)

	content := fmt.Sprintf("%s\n%s\nOverall Progress: %s", title, meta, progBar)

	return HeaderStyle.Width(m.width - 4).Render(content)
}

func (m DashboardModel) renderSidebar() string {
	var sections []string

	sections = append(sections, TitleStyle.Render("⚡ Requirement Dimensions"))

	dimensions := []struct {
		Name      string
		Score     int
		Rationale string
	}{
		{"Functional", m.Session.Scores.Functional, m.Session.Rationales.Functional},
		{"Structural", m.Session.Scores.Structural, m.Session.Rationales.Structural},
		{"Security",   m.Session.Scores.Security,   m.Session.Rationales.Security},
		{"Compliance", m.Session.Scores.Compliance, m.Session.Rationales.Compliance},
	}

	for _, d := range dimensions {
		prog := RenderProgressBar(18, d.Score)
		label := MetricLabelStyle.Render(d.Name)
		
		// Wrap rationale text
		rationale := d.Rationale
		if rationale == "" {
			rationale = "No feedback generated yet."
		}
		wrappedRationale := wrapText(rationale, 30)
		mutedRationale := lipgloss.NewStyle().Foreground(ColorMuted).Render(wrappedRationale)

		sections = append(sections, fmt.Sprintf("%s\n%s\n%s", label, prog, mutedRationale))
	}

	return strings.Join(sections, "\n\n")
}

func (m DashboardModel) renderMainChat() string {
	var content []string

	if m.isGenerating {
		content = append(content, TitleStyle.Render("✨ Final Asset Synthesis in Progress"))
		content = append(content, fmt.Sprintf("\n%s Running generative model downstream...", m.spinner.View()))
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render(m.genStatus))
		return strings.Join(content, "\n")
	}

	if m.isCompleted {
		content = append(content, TitleStyle.Render("🎉 Specification Complete!"))
		content = append(content, "\nAll requirement vectors have achieved 100% confidence and files have been generated.")
		content = append(content, "You can still edit raw facts to regenerate, or quit:\n")
		content = append(content, lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("  Press [G] to manually Regenerate files"))
		content = append(content, lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("  Press [E] to launch Editor & make modifications"))
		content = append(content, lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("  Press [Q] to Save & Exit CLI"))
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorMuted).Render(m.genStatus))
		return strings.Join(content, "\n")
	}

	// Interrogation Loop View
	content = append(content, TitleStyle.Render("💬 Conversation Timeline"))

	// Show last question from Oracle
	if m.Session.LastQuestion != "" && !m.loading {
		content = append(content, "\n"+QuestionStyle.Render("Architect's Question:"))
		content = append(content, wrapText(m.Session.LastQuestion, m.width-45))
	}

	// Show loading spinner
	if m.loading {
		content = append(content, fmt.Sprintf("\n%s Architectural Reasoning in progress. Calling AI API...", m.spinner.View()))
	} else {
		content = append(content, "\n"+InputPrefixStyle.Render("> ")+m.textInput.View())
	}

	return strings.Join(content, "\n")
}

func (m DashboardModel) renderFooter() string {
	var elements []string

	// Error Display
	if m.err != nil {
		elements = append(elements, lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render(fmt.Sprintf("⚠️  Error: %v", m.err)))
	}

	// Keybindings helper
	keys := []string{"Ctrl+C: Quit"}
	if !m.isCompleted {
		keys = append(keys, "Enter: Send", ":edit: Open full editor")
	}
	elements = append(elements, strings.Join(keys, "  |  "))

	return FooterStyle.Render(strings.Join(elements, "\n"))
}

// Simple text wrapping helper
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}
