package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/tui/shared"
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

	var body string
	if m.showViewer && m.isFullScreenViewer {
		styledChat := shared.MainPanelStyle.
			Width(m.width - 6).
			Height(bodyHeight).
			Render(mainChat)
		body = styledChat
	} else {
		// Truncate sidebar lines to bodyHeight to prevent vertical overflow pushing the footer offscreen
		sidebarLines := strings.Split(sidebar, "\n")
		if len(sidebarLines) > bodyHeight {
			sidebar = strings.Join(sidebarLines[:bodyHeight], "\n")
		}

		// Apply styles and dimensions
		styledSidebar := shared.SidebarStyle.
			Width(sidebarWidth).
			Height(bodyHeight).
			Render(sidebar)

		styledChat := shared.MainPanelStyle.
			Width(chatWidth).
			Height(bodyHeight).
			Render(mainChat)

		// Combine body horizontally
		body = lipgloss.JoinHorizontal(lipgloss.Top, styledSidebar, styledChat)
	}

	// Combine everything vertically
	fullView := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	return shared.DocStyle.Render(fullView)
}


func (m DashboardModel) renderHeader() string {
	title := " 🛠️  SynthSpec Solution Architect Dashboard "

	// Average Score Calculation
	avgScore := (m.Session.GetScores().Functional +
		m.Session.GetScores().Structural +
		m.Session.GetScores().Security +
		m.Session.GetScores().Compliance) / 4

	totalDurationStr := (time.Duration(m.Session.GetTotalDuration()) * time.Second).String()
	meta := fmt.Sprintf("Project: %s | Provider: %s | Model: %s | Tokens: %d | Cost: $%.4f | Time: %s",
		lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render(m.Session.GetProjectName()),
		strings.ToUpper(m.Session.GetProvider()),
		m.Session.GetModel(),
		m.Session.GetTotalTokens(),
		m.Session.GetEstimatedCost(),
		totalDurationStr,
	)

	progBar := shared.RenderProgressBar(40, avgScore)

	content := fmt.Sprintf("%s\n%s\nOverall Progress: %s", title, meta, progBar)

	return shared.HeaderStyle.Width(m.width - 4).Render(content)
}

func (m DashboardModel) renderSidebar() string {
	var sections []string

	sections = append(sections, shared.TitleStyle.Render("⚡ Requirement Dimensions"))

	dimensions := []struct {
		Name      string
		Score     int
		Rationale string
	}{
		{"Functional", m.Session.GetScores().Functional, m.Session.GetRationales().Functional},
		{"Structural", m.Session.GetScores().Structural, m.Session.GetRationales().Structural},
		{"Security", m.Session.GetScores().Security, m.Session.GetRationales().Security},
		{"Compliance", m.Session.GetScores().Compliance, m.Session.GetRationales().Compliance},
	}

	for _, d := range dimensions {
		prog := shared.RenderProgressBar(18, d.Score)
		label := shared.MetricLabelStyle.Render(d.Name)

		// Wrap rationale text
		rationale := d.Rationale
		if rationale == "" {
			rationale = "No feedback generated yet."
		}
		wrappedRationale := shared.WrapText(rationale, 30)
		rationaleLines := strings.Split(wrappedRationale, "\n")
		if len(rationaleLines) > 2 {
			wrappedRationale = strings.Join(rationaleLines[:2], "\n") + "..."
		}
		mutedRationale := lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(wrappedRationale)

		sections = append(sections, fmt.Sprintf("%s\n%s\n%s", label, prog, mutedRationale))
	}

	return strings.Join(sections, "\n\n")
}

func (m DashboardModel) renderThoughtBox() string {
	boxWidth := m.width - 45
	if boxWidth < 40 {
		boxWidth = 40
	}

	title := shared.ThoughtTitleStyle.Render("💭 Streaming Thought Box (Reasoning Tokens)")

	tokens := m.streamingTokens
	if tokens == "" {
		tokens = "Awaiting first token chunk..."
	}

	wrappedTokens := shared.WrapText(tokens, boxWidth-4)

	lines := strings.Split(wrappedTokens, "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	bodyContent := strings.Join(lines, "\n")

	return shared.ThoughtBoxStyle.Width(boxWidth).Render(title + "\n\n" + bodyContent)
}

func (m DashboardModel) renderMainChat() string {
	if m.showDiffViewer {
		return m.viewport.View()
	}
	if m.showViewer {
		return m.renderViewer()
	}
	if m.showUpdatePrompt {
		var content []string
		content = append(content, shared.TitleStyle.Render("📝 Add New Requirement / Modify Specification"))
		content = append(content, "\nEnter your new requirements or modifications below:")
		content = append(content, "\n"+m.updateInput.View())
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("(Press Enter to submit, Esc to cancel)"))
		return strings.Join(content, "\n")
	}
	return m.chatViewport.View()
}

// renderViewer draws the scrollable Markdown document viewer with a border, header, and footer.
func (m DashboardModel) renderViewer() string {
	if len(m.genFiles) == 0 {
		return "No files generated yet."
	}
	selectedFile := m.genFiles[m.selectedFileIdx]

	headerStyle := lipgloss.NewStyle().
		Foreground(shared.ColorAccent).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(shared.ColorMuted).
		Padding(0, 1)

	modeStr := "Split-Pane"
	if m.isFullScreenViewer {
		modeStr = "Full-Screen"
	}
	header := headerStyle.Render(fmt.Sprintf("📖 Viewing: %s (%s)", selectedFile, modeStr))

	footerStyle := lipgloss.NewStyle().
		Foreground(shared.ColorMuted).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(shared.ColorMuted).
		Padding(0, 1)

	scrollPercent := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)
	if m.viewport.AtBottom() {
		scrollPercent = "End"
	} else if m.viewport.AtTop() {
		scrollPercent = "Top"
	}

	footerText := fmt.Sprintf("Progress: %s  |  [Esc / q] Back  |  [f] Toggle Layout  |  [j / k] Scroll", scrollPercent)
	if m.isWaitingApproval {
		footerText = fmt.Sprintf("Progress: %s  |  [A / Enter] Approve & Resume  |  [E] Edit  |  [Esc / q] Back  |  [f] Toggle Layout  |  [j / k] Scroll", scrollPercent)
	}
	footer := footerStyle.Render(footerText)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n"+m.viewport.View(),
		footer,
	)
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
		wrappedBody := shared.WrapText(errBody, m.width-8)
		styledTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render("⚠️  " + errTitle)
		boxContent := fmt.Sprintf("%s\n%s", styledTitle, wrappedBody)

		styledBox := shared.ErrorBoxStyle.Width(m.width - 6).Render(boxContent)
		elements = append(elements, styledBox)
	}

	keys := []string{"Ctrl+C: Quit"}
	if m.err != nil {
		keys = append(keys, "Esc: Dismiss Error")
	} else if m.isGenerating {
		keys = append(keys, "q/Esc: Cancel", "f: Force Finish & Save")
	} else if m.isCompleted {
		keys = append(keys, "v/Enter: View file", "g: Regenerate & Verify", "u: Modify", "e: Editor")
	} else if m.showTextInput {
		keys = append(keys, "Enter: Send", "Ctrl+K: I Don't Know", "PgUp/PgDn: Scroll", ":edit: Open full editor", "Esc: Cancel")
	} else {
		keys = append(keys, "j/k/Arrows: Navigate", "Enter: Select", "Ctrl+K: I Don't Know", "PgUp/PgDn: Scroll", "g: Regenerate & Verify", "u: Modify", "e: Editor")
	}
	elements = append(elements, strings.Join(keys, "  |  "))

	return shared.FooterStyle.Render(strings.Join(elements, "\n"))
}

// updateChatViewport updates the dimensions and content of the interrogation chat viewport.
func (m *DashboardModel) updateChatViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	bodyHeight := m.height - headerHeight - footerHeight - 4
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	sidebarWidth := 34
	chatWidth := m.width - sidebarWidth - 8
	if chatWidth < 40 {
		chatWidth = 40
	}

	m.chatViewport.Width = chatWidth - 4
	m.chatViewport.Height = bodyHeight - 2

	if m.isGenerating {
		m.chatViewport.SetContent(m.renderGeneratingState())
	} else if m.isCompleted {
		m.chatViewport.SetContent(m.renderCompletedState())
	} else {
		m.chatViewport.SetContent(m.renderInterrogationState())
	}
}
