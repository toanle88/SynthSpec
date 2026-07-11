package dashboard

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/security"
	"github.com/toanle/synthspec/state"
)

// triggerRegeneration sets states and commands to begin a new specification file generation run.
func (m DashboardModel) triggerRegeneration() (tea.Model, tea.Cmd) {
	m.isGenerating = true
	m.isCompleted = false
	m.showScorecard = false
	m.genStatus = "Starting spec generation..."
	m.genPhase = "source"
	m.genChan = make(chan string, 10)
	m.genFileStatuses = make(map[string]string)
	m.genFileDetails = make(map[string]string)
	m.validatorLogs = nil
	for _, f := range m.genFiles {
		m.genFileStatuses[f] = "pending"
	}
	m.approvalChan = make(chan struct{})
	m.diffApprovalChan = make(chan struct{})
	m.isWaitingApproval = false
	m.isWaitingDiffApproval = false
	m.forceFinishChan = make(chan struct{})
	m.genStartTime = time.Now()
	m.genLogs = nil
	m.chatViewport.GotoTop()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelGen = cancel

	return m, tea.Batch(
		m.generateSpecsCmd(ctx),
		m.recvGenProgressCmd(),
		m.spinner.Tick,
		tickCmd(),
	)
}

// launchExternalEditor suspends Bubble Tea UI and runs the external system editor.
func (m DashboardModel) launchExternalEditor() (tea.Model, tea.Cmd) {
	editorCmd, tempPath, err := state.GetEditorCommand(m.Session.GetProjectName(), m.Session.GetFacts())
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.editorTempPath = tempPath
	return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// launchFileEditor suspends Bubble Tea UI and runs the external editor on an arbitrary generated file.
func (m DashboardModel) launchFileEditor(fileName string) (tea.Model, tea.Cmd) {
	dir := m.OutputDir
	if dir == "" {
		dir = filepath.Join(state.GetSessionDir(m.Session.GetProjectName()), "output")
	}
	filePath := filepath.Join(dir, fileName)
	editorCmd, err := state.GetFileEditorCommand(filePath)
	if err != nil {
		m.setError(err)
		return m, nil
	}
	m.isEditingFile = true
	return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return fileEditorFinishedMsg{err: err}
	})
}

// activateUpdatePrompt focuses the text input to enter manually updated requirements.
func (m DashboardModel) activateUpdatePrompt() (tea.Model, tea.Cmd) {
	m.showUpdatePrompt = true
	m.updateInput.Focus()
	m.updateInput.SetValue("")
	return m, nil
}

func (m DashboardModel) startOracleQuery(val string) (tea.Model, tea.Cmd) {
	if err := security.ScanForSecrets(val); err != nil {
		m.setError(err)
		return m, nil
	}
	_ = m.Session.SaveHistoryState()
	m.loading = true
	m.err = nil
	m.streamingTokens = ""
	m.thoughtBuffer = ""
	m.isStreaming = true
	m.isTyping = false
	// Recreate thoughtChan fresh every query so we never close an already-closed channel
	// and the streaming goroutine always writes into a live, empty channel.
	m.thoughtChan = make(chan string, 200)
	// Start a single throttled reader to process incoming tokens in batches.
	return m, tea.Batch(
		m.queryOracleCmd(val),
		m.recvThoughtCmd(),
	)
}

// getChoicesList formats standard and custom options to display on the interactive console list.
func (m DashboardModel) getChoicesList() []string {
	var list []string
	for i, c := range m.Session.GetLastChoices() {
		if i == 0 {
			list = append(list, "(Recommended) "+c)
		} else {
			list = append(list, c)
		}
	}
	list = append(list, "Let AI decide")
	list = append(list, "Custom user input...")
	return list
}

// updateSessionState updates the session structure fields with response metadata.
func (m *DashboardModel) updateSessionState(resp *gateway.OracleResponse) {
	m.Session.UpdateFacts(resp.Facts)
	m.Session.UpdateScores(resp.ConfidenceScores, m.Session.GetRationales())
	m.Session.UpdateScores(m.Session.GetScores(), resp.DimensionRationales)
	m.Session.SetInterrogationState(resp.NextQuestion, m.Session.GetLastChoices())
	m.Session.SetInterrogationState(m.Session.GetLastQuestion(), resp.NextChoices)
	m.Session.ClearGeneratedFiles()
	m.selectedChoiceIdx = 0
	m.showTextInput = len(m.Session.GetLastChoices()) == 0
}

// checkAndTriggerPostOracle performs checks to either initiate background document generation or queue context history compaction.
func (m DashboardModel) checkAndTriggerPostOracle(wasCompleted bool) (tea.Model, tea.Cmd) {
	var batchCmds []tea.Cmd
	if m.isCompleted && !wasCompleted {
		m.isGenerating = true
		m.genStatus = "Starting spec generation..."
		m.genPhase = "source"
		m.genChan = make(chan string, 10)
		m.genFileStatuses = make(map[string]string)
		m.genFileDetails = make(map[string]string)
		m.validatorLogs = nil
		for _, f := range m.genFiles {
			m.genFileStatuses[f] = "pending"
		}
		m.approvalChan = make(chan struct{})
		m.diffApprovalChan = make(chan struct{})
		m.isWaitingApproval = false
		m.isWaitingDiffApproval = false

		ctx, cancel := context.WithCancel(context.Background())
		m.cancelGen = cancel

		batchCmds = append(batchCmds, m.generateSpecsCmd(ctx), m.recvGenProgressCmd())
	} else if !m.isCompleted {
		m.loading = true
		batchCmds = append(batchCmds, m.pruneContextCmd())
	}
	return m, tea.Batch(batchCmds...)
}
