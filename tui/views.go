package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
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
		content = append(content, "\n"+TitleStyle.Render("📋 Engineering Quality Standards Check:"))
		content = append(content, m.renderStandardsGrid(m.width-45))
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
		if m.showScorecard {
			content = append(content, "\n"+TitleStyle.Render("📊 Final Architectural Quality Scorecard:"))
			content = append(content, m.renderStandardsGrid(m.width-45))
		}
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
		if m.showTextInput {
			content = append(content, "\n"+InputPrefixStyle.Render("> ")+m.textInput.View())
			content = append(content, "\n"+lipgloss.NewStyle().Foreground(ColorMuted).Render("(Press Esc to return to choices)"))
		} else {
			choices := m.getChoicesList()
			content = append(content, "\nSelect an option:")
			for i, choice := range choices {
				if i == m.selectedChoiceIdx {
					style := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
					content = append(content, style.Render(fmt.Sprintf("  ❯ %s", choice)))
				} else {
					content = append(content, fmt.Sprintf("    %s", choice))
				}
			}
		}
	}

	return strings.Join(content, "\n")
}

func (m DashboardModel) renderFooter() string {
	var elements []string

	// Error Display
	if m.err != nil {
		var errTitle string
		var errBody string

		if apiErr, ok := m.err.(*gateway.APIError); ok {
			errTitle = fmt.Sprintf("API REQUEST FAILED (%d)", apiErr.StatusCode)
			errBody = apiErr.Message
			if apiErr.RetryAfter != "" {
				errBody += fmt.Sprintf("\nSuggested retry delay: %s", apiErr.RetryAfter)
			}
		} else {
			errTitle = "SYSTEM ERROR"
			errBody = m.err.Error()
		}

		// Wrap the error body to the dashboard width to prevent stretching the layout
		wrappedBody := wrapText(errBody, m.width-8)
		styledTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render("⚠️  " + errTitle)
		boxContent := fmt.Sprintf("%s\n%s", styledTitle, wrappedBody)

		styledBox := ErrorBoxStyle.Width(m.width - 6).Render(boxContent)
		elements = append(elements, styledBox)
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
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}

		// Find leading spaces to preserve indentation
		indent := ""
		for _, c := range line {
			if c == ' ' || c == '\t' {
				indent += string(c)
			} else {
				break
			}
		}

		trimmed := strings.TrimSpace(line)
		words := strings.Fields(trimmed)
		if len(words) == 0 {
			result = append(result, indent)
			continue
		}

		currentLine := indent + words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) > width {
				result = append(result, currentLine)
				currentLine = indent + word
			} else {
				currentLine += " " + word
			}
		}
		result = append(result, currentLine)
	}

	return strings.Join(result, "\n")
}

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // Vibrant Red
)

func (m DashboardModel) getStandardStatus(std config.Standard) (string, lipgloss.Style) {
	if m.showScorecard {
		score, found := m.complianceScores[std.ID]
		if !found {
			return "🔴 N/A", StyleError
		}
		if score >= std.MinScore {
			return fmt.Sprintf("🟢 %d%%", score), StyleSuccess
		} else if score > 0 {
			return fmt.Sprintf("🟡 %d%%", score), StyleWarning
		} else {
			return fmt.Sprintf("🔴 %d%%", score), StyleError
		}
	}

	if !m.isGenerating {
		return "⏳ Pending", StyleMuted
	}

	// Check if this standard targets the file currently being processed
	for _, tf := range std.TargetFiles {
		if strings.Contains(m.genStatus, tf) {
			if strings.Contains(m.genStatus, "Auditing") || strings.Contains(m.genStatus, "Refining") || strings.Contains(m.genStatus, "failed") {
				return "🔄 Auditing", StyleInfo
			}
			return "⏳ Building", StyleMuted
		}
	}

	// Determine if the standard's target file is in the past
	files := []string{
		"01_prd_functional.md",
		"02_system_architecture.md",
		"03_security_threat_model.md",
		"04_openapi_contract.yaml",
		"05_engineering_backlog.json",
	}

	currentFileIdx := -1
	for i, f := range files {
		if strings.Contains(m.genStatus, f) {
			currentFileIdx = i
			break
		}
	}

	if currentFileIdx == -1 {
		if strings.Contains(m.genStatus, "successfully") || strings.Contains(m.genStatus, "Compiling") || strings.Contains(m.genStatus, "audited") {
			return "🟢 Verified", StyleSuccess
		}
		return "⏳ Pending", StyleMuted
	}

	isPast := true
	for _, tf := range std.TargetFiles {
		stdFileIdx := -1
		for i, f := range files {
			if f == tf {
				stdFileIdx = i
				break
			}
		}
		if stdFileIdx >= currentFileIdx {
			isPast = false
		}
	}

	if isPast {
		return "🟢 Verified", StyleSuccess
	}

	return "⏳ Pending", StyleMuted
}

func (m DashboardModel) renderStandardsGrid(width int) string {
	var leftCol []string
	var rightCol []string

	half := (len(m.standards) + 1) / 2
	for i, std := range m.standards {
		statusText, style := m.getStandardStatus(std)

		styledLabel := lipgloss.NewStyle().Foreground(ColorText).Render(std.Name)
		styledStatus := style.Bold(true).Render(statusText)

		padding := 28 - len(std.Name)
		if padding < 1 {
			padding = 1
		}
		item := fmt.Sprintf("  %s:%s%s", styledLabel, strings.Repeat(" ", padding), styledStatus)

		if i < half {
			leftCol = append(leftCol, item)
		} else {
			rightCol = append(rightCol, item)
		}
	}

	leftBlock := strings.Join(leftCol, "\n")
	rightBlock := strings.Join(rightCol, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftBlock,
		"       ", // spacer
		rightBlock,
	)
}

