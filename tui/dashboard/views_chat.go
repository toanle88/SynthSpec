package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

func (m DashboardModel) renderGeneratingState() string {
	var content []string
	content = append(content, shared.TitleStyle.Render("✨ Final Asset Synthesis in Progress"))
	switch m.genPhase {
	case "source":
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("Phase: Source document lock-in"))
	case "parallel":
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("Phase: Parallel downstream generation"))
		if len(m.genFiles) > 1 {
			content = append(content, lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(fmt.Sprintf("Fan-out active: %d downstream documents running in parallel.", len(m.genFiles)-1)))
		}
	}
	if m.isWaitingApproval {
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorWarning).Bold(true).Render("⏸️  THE DOMAIN APPROVAL GATE ACTIVE"))
		content = append(content, "The source domain model has been generated and validated.")
		content = append(content, "Please review the file and approve it to proceed with downstream parallel generation:\n")
		content = append(content, lipgloss.NewStyle().Foreground(shared.ColorSuccess).Bold(true).Render("  Press [V] to View 01_domain_model_use_cases.md"))
		content = append(content, lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("  Press [E] to Edit 01_domain_model_use_cases.md"))
		content = append(content, lipgloss.NewStyle().Foreground(shared.ColorSuccess).Bold(true).Render("  Press [A] or [Enter] to Approve and Resume Downstream Synthesis"))
	} else {
		content = append(content, fmt.Sprintf("\n%s Running generative model downstream...", m.spinner.View()))
	}
	content = append(content, "\n"+shared.TitleStyle.Render("📂 Document Synthesis Progress:"))
	content = append(content, m.renderFileProgressList())
	content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorSuccess).Bold(true).Render("Status: "+m.genStatus))
	content = append(content, "\n"+shared.TitleStyle.Render("📋 Engineering Quality Standards Check:"))
	content = append(content, m.renderStandardsGrid())
	if len(m.validatorLogs) > 0 {
		content = append(content, "\n"+shared.TitleStyle.Render("💻 Validator Live Console Logs:"))
		boxContent := strings.Join(m.validatorLogs, "\n")
		styledLogs := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a1a1aa")).
			Background(lipgloss.Color("#18181b")).
			Padding(1, 2).
			Width(m.width - 45).
			Render(boxContent)
		content = append(content, styledLogs)
	}
	return strings.Join(content, "\n")
}

func (m DashboardModel) renderCompletedState() string {
	var content []string
	content = append(content, shared.TitleStyle.Render("🎉 Specification Complete!"))
	if m.genPhase == "parallel" {
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("Last phase: Parallel downstream generation"))
	}
	content = append(content, "\nAll requirement vectors have achieved 100% confidence and files have been generated.")
	content = append(content, "You can still edit raw facts to regenerate, or quit:\n")
	content = append(content, lipgloss.NewStyle().Foreground(shared.ColorSuccess).Bold(true).Render("  Press [G] to manually Regenerate & Verify consistency"))
	content = append(content, lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("  Press [U] to Add new requirements / Modify specifications"))
	content = append(content, lipgloss.NewStyle().Foreground(shared.ColorInfo).Bold(true).Render("  Press [E] to launch Editor & make modifications"))
	content = append(content, lipgloss.NewStyle().Foreground(shared.ColorWarning).Bold(true).Render("  Press [Q] to Save & Exit CLI"))
	content = append(content, "\n"+shared.TitleStyle.Render("📂 Document Synthesis Status:"))
	content = append(content, m.renderFileProgressList())
	content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(m.genStatus))
	if m.showScorecard {
		content = append(content, "\n"+shared.TitleStyle.Render("📊 Final Architectural Quality Scorecard:"))
		content = append(content, m.renderStandardsGrid())
	}
	return strings.Join(content, "\n")
}

func (m DashboardModel) renderInterrogationState() string {
	var content []string
	content = append(content, shared.TitleStyle.Render("💬 Conversation Timeline"))

	if m.Session.LastQuestion != "" && !m.loading && !m.isStreaming {
		content = append(content, "\n"+shared.QuestionStyle.Render("Architect's Question:"))
		content = append(content, shared.WrapText(m.Session.LastQuestion, m.width-45))
	}

	if m.loading || m.isStreaming {
		if m.loading {
			content = append(content, fmt.Sprintf("\n%s Architectural Reasoning in progress. Calling AI API...", m.spinner.View()))
		} else {
			content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorSuccess).Bold(true).Render("✓ Response received — streaming thought tokens..."))
		}
		content = append(content, "\n"+m.renderThoughtBox())
	} else if m.showTextInput {
		content = append(content, "\n"+m.textInput.View())
		content = append(content, "\n"+lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("(Press Esc to return to choices)"))
	} else {
		choices := m.getChoicesList()
		content = append(content, "\nSelect an option:")
		for i, choice := range choices {
			if i == m.selectedChoiceIdx {
				style := lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
				content = append(content, style.Render(fmt.Sprintf("  ❯ %s", choice)))
			} else {
				content = append(content, fmt.Sprintf("    %s", choice))
			}
		}
	}
	return strings.Join(content, "\n")
}
