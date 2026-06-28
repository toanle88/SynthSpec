package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/state"
)

// Msg Types
type oracleResultMsg struct {
	resp *gateway.OracleResponse
	err  error
}

type editorFinishedMsg struct {
	err error
}

type genProgressMsg string
type genFinishedMsg struct {
	err error
}

type contextPruneResultMsg struct {
	pruned bool
	err    error
}

// DashboardModel represents the TUI state
type DashboardModel struct {
	Session   *state.Session
	Gateway   gateway.Gateway
	OutputDir string

	textInput textinput.Model
	spinner   spinner.Model
	loading   bool
	err       error

	// Layout sizes
	width  int
	height int

	// Editor state
	editorTempPath string

	// Generation state
	isCompleted     bool
	isGenerating    bool
	genStatus       string
	genPhase        string
	genChan         chan string
	genFiles        []string
	genFileStatuses map[string]string
	genFileDetails  map[string]string

	// Compliance scorecard state
	standards        []config.Standard
	complianceScores map[string]int
	showScorecard    bool

	// Choice selection state
	selectedChoiceIdx int
	showTextInput     bool

	// External validator logs
	validatorLogs []string

	// Update requirement state
	showUpdatePrompt bool
	updateInput      textinput.Model
	isCLIUpdateMode  bool
}

func NewDashboardModel(sess *state.Session, gw gateway.Gateway, outputDir string) DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	ti := textinput.New()
	ti.Placeholder = "Type your answer here, or ':edit' to open in full editor..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 60

	// Check if already 100% completed
	completed := checkCompletion(sess.Scores)

	standards, _ := config.LoadStandards()

	// If already completed and output meta exists, try to load scores
	complianceScores := make(map[string]int)
	showScorecard := false
	if completed {
		dir := outputDir
		if dir == "" {
			dir = filepath.Join(state.GetSessionDir(sess.ProjectName), "output")
		}
		metaPath := filepath.Join(dir, ".synthspec-meta.json")
		if metaBytes, readErr := os.ReadFile(metaPath); readErr == nil {
			var meta struct {
				ComplianceSummary map[string]int `json:"compliance_summary"`
			}
			if jsonErr := json.Unmarshal(metaBytes, &meta); jsonErr == nil && len(meta.ComplianceSummary) > 0 {
				complianceScores = meta.ComplianceSummary
				showScorecard = true
			}
		}
	}

	showTextInput := len(sess.LastChoices) == 0

	templates, _ := config.LoadTemplates()
	var genFiles []string
	for _, t := range templates {
		genFiles = append(genFiles, t.FileName)
	}
	genFileStatuses := make(map[string]string)
	genFileDetails := make(map[string]string)
	for _, f := range genFiles {
		genFileStatuses[f] = "pending"
	}

	ui := textinput.New()
	ui.Placeholder = "Type new requirements or modifications here..."
	ui.CharLimit = 2000
	ui.Width = 60

	return DashboardModel{
		Session:           sess,
		Gateway:           gw,
		OutputDir:         outputDir,
		textInput:         ti,
		updateInput:       ui,
		spinner:           s,
		isCompleted:       completed,
		standards:         standards,
		complianceScores:  complianceScores,
		showScorecard:     showScorecard,
		selectedChoiceIdx: 0,
		showTextInput:     showTextInput,
		genFiles:          genFiles,
		genFileStatuses:   genFileStatuses,
		genFileDetails:    genFileDetails,
	}
}

func (m *DashboardModel) StartWithUpdatePrompt() {
	m.showUpdatePrompt = true
	m.isCLIUpdateMode = true
	m.updateInput.Focus()
}

func checkCompletion(scores gateway.ConfidenceScores) bool {
	return scores.Functional >= 100 &&
		scores.Structural >= 100 &&
		scores.Security >= 100 &&
		scores.Compliance >= 100
}

func (m DashboardModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)

	// Bootstrapping: If history is empty and last question is empty, query Oracle first
	if len(m.Session.History) == 0 && m.Session.LastQuestion == "" {
		cmds = append(cmds, m.queryOracleCmd(""))
	}

	return tea.Batch(cmds...)
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Prevent typing while loading or generating
		if m.loading || m.isGenerating {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.showUpdatePrompt {
				val := strings.TrimSpace(m.updateInput.Value())
				if val == "" {
					return m, nil
				}
				m.updateInput.SetValue("")
				m.showUpdatePrompt = false
				m.isCompleted = false
				m.loading = true
				m.err = nil
				return m, m.queryOracleCmd("I have a new requirement/change: " + val)
			}

			if m.isCompleted {
				// On completion screen, Enter does nothing. Use explicit keys.
				return m, nil
			}

			if m.showTextInput {
				val := strings.TrimSpace(m.textInput.Value())
				if val == "" {
					return m, nil
				}

				m.textInput.SetValue("")

				// Handle Editor command direct typing
				if val == ":edit" {
					// Run Editor Subprocess
					editorCmd, tempPath, err := state.GetEditorCommand(m.Session.ProjectName, m.Session.Facts)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.editorTempPath = tempPath
					// Suspend Bubble Tea and run editor
					return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
						return editorFinishedMsg{err: err}
					})
				}

				// Regular answer submission
				m.loading = true
				m.err = nil
				return m, m.queryOracleCmd(val)
			} else {
				choices := m.getChoicesList()
				selected := choices[m.selectedChoiceIdx]

				if selected == "Custom user input..." {
					m.showTextInput = true
					m.textInput.Focus()
					m.textInput.SetValue("")
					return m, nil
				}

				var val string
				if selected == "Let AI decide" {
					val = "Let the AI decide based on current facts and context."
				} else {
					// Extract raw choice
					val = m.Session.LastChoices[m.selectedChoiceIdx]
				}

				m.loading = true
				m.err = nil
				return m, m.queryOracleCmd(val)
			}

		case tea.KeyUp, tea.KeyPgUp:
			if !m.showTextInput && !m.showUpdatePrompt {
				choices := m.getChoicesList()
				m.selectedChoiceIdx--
				if m.selectedChoiceIdx < 0 {
					m.selectedChoiceIdx = len(choices) - 1
				}
			}

		case tea.KeyDown, tea.KeyPgDown:
			if !m.showTextInput && !m.showUpdatePrompt {
				choices := m.getChoicesList()
				m.selectedChoiceIdx++
				if m.selectedChoiceIdx >= len(choices) {
					m.selectedChoiceIdx = 0
				}
			}

		case tea.KeyEsc:
			if m.showUpdatePrompt {
				m.showUpdatePrompt = false
				m.updateInput.Blur()
				if m.isCLIUpdateMode {
					return m, tea.Quit
				}
				return m, nil
			}
			if m.showTextInput && len(m.Session.LastChoices) > 0 {
				m.showTextInput = false
				m.textInput.Blur()
			}

		case tea.KeyRunes:
			if m.showUpdatePrompt {
				// Don't process other keybindings if update input is focused
				break
			}
			if m.isCompleted {
				key := string(msg.Runes)
				switch strings.ToLower(key) {
				case "g":
					// Trigger Specs Generation
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
					return m, tea.Batch(
						m.generateSpecsCmd(),
						m.recvGenProgressCmd(),
					)
				case "e":
					// Launch Editor
					editorCmd, tempPath, err := state.GetEditorCommand(m.Session.ProjectName, m.Session.Facts)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.editorTempPath = tempPath
					return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
						return editorFinishedMsg{err: err}
					})
				case "u":
					m.showUpdatePrompt = true
					m.updateInput.Focus()
					m.updateInput.SetValue("")
					return m, nil
				case "q":
					return m, tea.Quit
				}
			} else if !m.showTextInput {
				key := string(msg.Runes)
				switch key {
				case "k":
					choices := m.getChoicesList()
					m.selectedChoiceIdx--
					if m.selectedChoiceIdx < 0 {
						m.selectedChoiceIdx = len(choices) - 1
					}
				case "j":
					choices := m.getChoicesList()
					m.selectedChoiceIdx++
					if m.selectedChoiceIdx >= len(choices) {
						m.selectedChoiceIdx = 0
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case oracleResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Update session details
		m.Session.Facts = msg.resp.Facts
		m.Session.Scores = msg.resp.ConfidenceScores
		m.Session.Rationales = msg.resp.DimensionRationales
		m.Session.LastQuestion = msg.resp.NextQuestion
		m.Session.LastChoices = msg.resp.NextChoices
		m.Session.GeneratedFiles = nil
		m.selectedChoiceIdx = 0
		m.showTextInput = len(m.Session.LastChoices) == 0

		// If user entered answer, record history (except boot queries)
		if len(m.Session.History) > 0 || m.textInput.Value() != "" {
			// We track in Update before user clears it, but we cleared it already.
			// Let's pass the text input to queryOracleCmd, which we did. We'll reconstruct the turn.
		}

		// Save session progress
		m.Session.Save()
		wasCompleted := m.isCompleted
		m.isCompleted = checkCompletion(m.Session.Scores)

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
			batchCmds = append(batchCmds, m.generateSpecsCmd(), m.recvGenProgressCmd())
		} else if !m.isCompleted {
			m.loading = true
			batchCmds = append(batchCmds, m.pruneContextCmd())
		}

		return m, tea.Batch(batchCmds...)

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("editor failed: %w", msg.err)
			return m, nil
		}

		// Read back edited facts
		editedFacts, err := state.ReadBackEditedFacts(m.editorTempPath)
		if err != nil {
			m.err = fmt.Errorf("failed to read back edited requirements: %w", err)
			return m, nil
		}

		m.Session.Facts = editedFacts
		m.Session.GeneratedFiles = nil
		m.Session.Save()

		// Re-trigger Oracle query to evaluate updated facts
		m.loading = true
		m.err = nil
		return m, m.queryOracleCmd("Requirements updated manually via editor.")

	case genProgressMsg:
		var ev generator.ProgressEvent
		if err := json.Unmarshal([]byte(msg), &ev); err == nil {
			if ev.Status == "started" {
				if ev.Phase != "" {
					m.genPhase = ev.Phase
				}
				m.genFiles = strings.Split(ev.Details, ",")
				m.genFileStatuses = make(map[string]string)
				m.genFileDetails = make(map[string]string)
				m.validatorLogs = nil
				for _, f := range m.genFiles {
					m.genFileStatuses[f] = "pending"
				}
			} else if ev.File != "" {
				if m.genFileStatuses == nil {
					m.genFileStatuses = make(map[string]string)
				}
				if m.genFileDetails == nil {
					m.genFileDetails = make(map[string]string)
				}
				m.genFileStatuses[ev.File] = ev.Status
				m.genFileDetails[ev.File] = ev.Details
			}
			if ev.ValLogs != "" {
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
			if ev.Message != "" {
				m.genStatus = ev.Message
			}
		} else {
			m.genStatus = string(msg)
		}
		return m, m.recvGenProgressCmd()

	case genFinishedMsg:
		m.isGenerating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.genStatus = "All specifications synthesized successfully!"
			m.Session.Save() // Save final state

			// Try to read .synthspec-meta.json to populate complianceScores
			dir := m.OutputDir
			if dir == "" {
				dir = filepath.Join(state.GetSessionDir(m.Session.ProjectName), "output")
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
		return m, nil

	case contextPruneResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("context pruning failed: %w", msg.err)
		} else if msg.pruned {
			m.err = fmt.Errorf("conversation summarized to fit context limit")
		}
		return m, nil
	}

	// Update text input
	if !m.isCompleted && !m.loading && m.showTextInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showUpdatePrompt && !m.loading {
		m.updateInput, cmd = m.updateInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// Background commands
func (m DashboardModel) queryOracleCmd(latestInput string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// If user answer was provided, append it to history beforehand
		if latestInput != "" && latestInput != "Requirements updated manually via editor." {
			m.Session.AddTurn(latestInput, m.Session.LastQuestion, m.Session.TotalTokensUsed, m.Session.TotalTokensUsed)
		}

		resp, err := m.Gateway.QueryOracle(ctx, m.Session.Facts, m.Session.History, latestInput)
		if err != nil {
			return oracleResultMsg{err: err}
		}

		// Update tokens in session (will be saved in Update msg handler)
		if latestInput != "" && latestInput != "Requirements updated manually via editor." {
			// Back-fill actual assistant response
			m.Session.History[len(m.Session.History)-1].Content = resp.NextQuestion
		}
		m.Session.TotalTokensUsed = resp.TokensPrompt + resp.TokensCompletion

		return oracleResultMsg{resp: resp}
	}
}

// Receives generator logs asynchronously
func (m DashboardModel) recvGenProgressCmd() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.genChan
		if !ok {
			return nil
		}
		return genProgressMsg(progress)
	}
}

// Background command to run generation sequentially
func (m DashboardModel) generateSpecsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := generator.Generate(ctx, m.Gateway, m.Session, m.OutputDir, m.genChan)
		return genFinishedMsg{err: err}
	}
}

func (m DashboardModel) pruneContextCmd() tea.Cmd {
	return func() tea.Msg {
		pruned, err := m.Session.CheckAndPruneContext(context.Background(), m.Gateway)
		return contextPruneResultMsg{pruned: pruned, err: err}
	}
}

func (m DashboardModel) getChoicesList() []string {
	var list []string
	for i, c := range m.Session.LastChoices {
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
