package dashboard

import (
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
		return m.handleOracleResult(msg)

	case editorFinishedMsg:
		return m.handleEditorFinished(msg)

	case fileEditorFinishedMsg:
		return m.handleUpdateFileEditorFinishedMsg(msg)

	case genProgressMsg:
		return m.handleGenProgress(msg)

	case genFinishedMsg:
		return m.handleGenFinished(msg)

	case contextPruneResultMsg:
		return m.handleContextPruneResult(msg)

	case initQueryMsg:
		return m.startOracleQuery("")

	case thoughtTokenMsg:
		return m.handleUpdateThoughtTokenMsg(msg)

	case typingTickMsg:
		return m.handleUpdateTypingTickMsg()

	case streamDoneMsg:
		m.isStreaming = false
		if !m.isTyping {
			m.updateChatViewport()
			m.chatViewport.GotoBottom()
		}

	case tea.MouseMsg:
		return m.handleUpdateMouseMsg(msg)
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
