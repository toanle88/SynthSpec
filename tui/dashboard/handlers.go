package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
)

func (m DashboardModel) handleUpdateFileEditorFinishedMsg(msg fileEditorFinishedMsg) (tea.Model, tea.Cmd) {
	m.isEditingFile = false
	if msg.err != nil {
		m.setError(fmt.Errorf("file editor failed: %w", msg.err))
	}
	if m.isWaitingApproval && m.showViewer {
		return m.openFileViewer()
	}
	return m, nil
}

func (m DashboardModel) handleUpdateThoughtTokenMsg(msg thoughtTokenMsg) (tea.Model, tea.Cmd) {
	m.thoughtBuffer += string(msg)
	var tickCmd tea.Cmd
	if !m.isTyping && len(m.thoughtBuffer) > 0 {
		m.isTyping = true
		tickCmd = tea.Tick(35*time.Millisecond, func(t time.Time) tea.Msg {
			return typingTickMsg{}
		})
	}
	return m, tea.Batch(m.recvThoughtCmd(), tickCmd)
}

func (m DashboardModel) handleUpdateTypingTickMsg() (tea.Model, tea.Cmd) {
	if len(m.thoughtBuffer) > 0 {
		runes := []rune(m.thoughtBuffer)
		rChunkSize := len(runes) / 60
		if rChunkSize < 1 {
			rChunkSize = 1
		}
		if rChunkSize > len(runes) {
			rChunkSize = len(runes)
		}
		chunk := string(runes[:rChunkSize])
		m.streamingTokens += chunk
		m.thoughtBuffer = string(runes[rChunkSize:])

		m.updateChatViewport()
		m.chatViewport.GotoBottom()

		return m, tea.Tick(35*time.Millisecond, func(t time.Time) tea.Msg {
			return typingTickMsg{}
		})
	}
	m.isTyping = false
	if !m.isStreaming {
		m.updateChatViewport()
		m.chatViewport.GotoBottom()
	}
	return m, nil
}

// handleOracleResult processes updates returned by the Oracle LLM model, saving session state and checking for completeness.
func (m DashboardModel) handleOracleResult(msg oracleResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.setError(msg.err)
		return m, nil
	}

	m.updateSessionState(msg.resp)

	if err := m.Session.Save(); err != nil {
		logger.Log("session save failed after oracle result: %v", err)
	}
	wasCompleted := m.isCompleted
	m.isCompleted = checkCompletion(m.Session.GetScores())

	return m.checkAndTriggerPostOracle(wasCompleted)
}

// handleEditorFinished reads back edited requirement documents when external editor processes terminate.
func (m DashboardModel) handleEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.setError(fmt.Errorf("editor failed: %w", msg.err))
		return m, nil
	}

	editedFacts, err := state.ReadBackEditedFacts(m.editorTempPath)
	if err != nil {
		m.setError(fmt.Errorf("failed to read back edited requirements: %w", err))
		return m, nil
	}

	m.Session.UpdateFacts(editedFacts)
	m.Session.ClearGeneratedFiles()
	if err := m.Session.Save(); err != nil {
		logger.Log("session save failed after editor: %v", err)
	}

	return m.startOracleQuery(manualUpdateMsg)
}

// handleGenProgress processes incoming specification file generation progress notifications.
func (m DashboardModel) handleGenProgress(msg genProgressMsg) (tea.Model, tea.Cmd) {
	var ev generator.ProgressEvent
	if err := json.Unmarshal([]byte(msg), &ev); err != nil {
		m.genStatus = string(msg)
		return m, m.recvGenProgressCmd()
	}

	if ev.Status == "started" {
		m.handleGenProgressStart(ev)
	} else if ev.File != "" {
		m.handleGenProgressFile(ev)
	}

	if ev.ValLogs != "" {
		m.handleGenProgressLogs(ev)
	}

	if ev.Message != "" {
		m.genStatus = ev.Message
	}

	if ev.Status == "waiting_approval" {
		m.isWaitingApproval = true
		m.selectedFileIdx = 0
		model, cmd := m.openFileViewer()
		m = model.(DashboardModel)
		return m, tea.Batch(cmd, m.recvGenProgressCmd())
	}

	if ev.Status == "waiting_diff_approval" {
		var diffs []domain.FileDiff
		if err := json.Unmarshal([]byte(ev.Details), &diffs); err == nil {
			model, cmd := m.startDiffApproval(diffs)
			m = model.(DashboardModel)
			return m, tea.Batch(cmd, m.recvGenProgressCmd())
		}
	}

	return m, m.recvGenProgressCmd()
}

// handleGenProgressStart initializes the TUI metadata maps when document synthesis first starts.
func (m *DashboardModel) handleGenProgressStart(ev generator.ProgressEvent) {
	if ev.Phase != "" {
		m.genPhase = ev.Phase
	}
	m.genFiles = strings.Split(ev.Details, ",")
	if m.genFileStatuses == nil {
		m.genFileStatuses = make(map[string]string)
	}
	if m.genFileDetails == nil {
		m.genFileDetails = make(map[string]string)
	}
	m.validatorLogs = nil
	for _, f := range m.genFiles {
		if _, exists := m.genFileStatuses[f]; !exists {
			m.genFileStatuses[f] = "pending"
		}
	}
}

// handleGenProgressFile updates status information and evaluation details for individual target assets.
func (m *DashboardModel) handleGenProgressFile(ev generator.ProgressEvent) {
	if m.genFileStatuses == nil {
		m.genFileStatuses = make(map[string]string)
	}
	if m.genFileDetails == nil {
		m.genFileDetails = make(map[string]string)
	}
	m.genFileStatuses[ev.File] = ev.Status
	m.genFileDetails[ev.File] = ev.Details
}

// handleGenProgressLogs appends real-time validation logs and limits memory capacity to the most recent 10 lines.
func (m *DashboardModel) handleGenProgressLogs(ev generator.ProgressEvent) {
	lines := strings.Split(ev.ValLogs, "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			m.validatorLogs = append(m.validatorLogs, l)
		}
	}
	if len(m.validatorLogs) > 10 {
		m.validatorLogs = m.validatorLogs[len(m.validatorLogs)-10:]
	}
}

// handleGenFinished saves final status and parses scorecard performance statistics when the asset synthesis ends.
func (m DashboardModel) handleGenFinished(msg genFinishedMsg) (tea.Model, tea.Cmd) {
	m.isGenerating = false
	m.isCompleted = checkCompletion(m.Session.GetScores())
	if msg.err != nil {
		if msg.err == context.Canceled || strings.Contains(msg.err.Error(), "context canceled") {
			m.genStatus = "Specification generation cancelled."
		} else {
			m.setError(msg.err)
		}
	} else {
		m.genStatus = "All specifications synthesized successfully!"
		// Mark all successfully generated files as done
		for _, f := range m.genFiles {
			if m.genFileStatuses[f] != "failed" {
				m.genFileStatuses[f] = "done"
			}
		}

		if err := m.Session.Save(); err != nil {
			logger.Log("session save failed after generation: %v", err)
		}

		dir := m.OutputDir
		if dir == "" {
			dir = filepath.Join(state.GetSessionDir(m.Session.GetProjectName()), "output")
		}
		metaPath := filepath.Join(dir, ".synthspec-meta.json")
		if metaBytes, readErr := os.ReadFile(metaPath); readErr == nil {
			var meta struct {
				ComplianceSummary map[string]int `json:"compliance_summary"`
			}
			if jsonErr := json.Unmarshal(metaBytes, &meta); jsonErr == nil {
				m.complianceScores = meta.ComplianceSummary
				m.showScorecard = true
				m.genStatus = "All specifications synthesized and audited successfully!"
			}
		}
	}
	m.chatViewport.GotoTop()
	return m, nil
}

// handleContextPruneResult processes the outcome of Oracle conversation history context compaction.
func (m DashboardModel) handleContextPruneResult(msg contextPruneResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.setError(fmt.Errorf("context pruning failed: %w", msg.err))
	} else if msg.pruned {
		m.genStatus = "Conversation summarized to fit context limit."
	}
	return m, nil
}
