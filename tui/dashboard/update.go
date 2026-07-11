package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Always handle spinner ticks to prevent the animation tick loop from breaking
	if _, ok := msg.(spinner.TickMsg); ok {
		m.spinner, cmd = m.spinner.Update(msg)
		if m.showViewer {
			m.viewport, _ = m.viewport.Update(msg)
			return m, cmd
		}
		m.updateChatViewport()
		return m, cmd
	}

	if m.showDiffViewer {
		return m.handleDiffViewerUpdate(msg)
	}

	if m.showViewer {
		return m.handleViewerUpdate(msg)
	}

	switch msg := msg.(type) {
	case timerTickMsg:
		if m.isGenerating {
			cmds = append(cmds, tickCmd())
		}
		m.updateChatViewport()
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		var keyCmd tea.Cmd
		var model tea.Model
		model, keyCmd = m.handleUpdateKeyMsg(msg)
		m = model.(DashboardModel)
		if keyCmd != nil {
			cmds = append(cmds, keyCmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateChatViewport()

	case oracleResultMsg:
		model, cmd := m.handleOracleResult(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case editorFinishedMsg:
		model, cmd := m.handleEditorFinished(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case fileEditorFinishedMsg:
		model, cmd := m.handleUpdateFileEditorFinishedMsg(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case genProgressMsg:
		model, cmd := m.handleGenProgress(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case genFinishedMsg:
		model, cmd := m.handleGenFinished(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case contextPruneResultMsg:
		model, cmd := m.handleContextPruneResult(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case initQueryMsg:
		model, cmd := m.startOracleQuery("")
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case thoughtTokenMsg:
		model, cmd := m.handleUpdateThoughtTokenMsg(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case typingTickMsg:
		model, cmd := m.handleUpdateTypingTickMsg()
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case streamDoneMsg:
		m.isStreaming = false
		if !m.isTyping {
			m.updateChatViewport()
			m.chatViewport.GotoBottom()
		}

	case tea.MouseMsg:
		model, cmd := m.handleUpdateMouseMsg(msg)
		m = model.(DashboardModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if !m.isCompleted && !m.loading && m.showTextInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showUpdatePrompt && !m.loading {
		m.updateInput, cmd = m.updateInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.updateChatViewport()
	return m, tea.Batch(cmds...)
}

type timerTickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return timerTickMsg(t)
	})
}
