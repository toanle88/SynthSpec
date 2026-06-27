package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
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

// DashboardModel represents the TUI state
type DashboardModel struct {
	Session         *state.Session
	Gateway         gateway.Gateway
	OutputDir       string
	
	textInput       textinput.Model
	spinner         spinner.Model
	loading         bool
	err             error
	
	// Layout sizes
	width           int
	height          int
	
	// Editor state
	editorTempPath  string
	
	// Generation state
	isCompleted     bool // When confidence is 100%
	isGenerating    bool
	genStatus       string
	genChan         chan string
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

	return DashboardModel{
		Session:     sess,
		Gateway:     gw,
		OutputDir:   outputDir,
		textInput:   ti,
		spinner:     s,
		isCompleted: completed,
	}
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
			if m.isCompleted {
				// On completion screen, Enter does nothing. Use explicit keys.
				return m, nil
			}

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

		case tea.KeyRunes:
			if m.isCompleted {
				key := string(msg.Runes)
				switch strings.ToLower(key) {
				case "g":
					// Trigger Specs Generation
					m.isGenerating = true
					m.genStatus = "Starting spec generation..."
					m.genChan = make(chan string, 10)
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
				case "q":
					return m, tea.Quit
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
			m.genChan = make(chan string, 10)
			batchCmds = append(batchCmds, m.generateSpecsCmd(), m.recvGenProgressCmd())
		}

		// Context Pruning check
		pruned, pruneErr := m.Session.CheckAndPruneContext(context.Background(), m.Gateway)
		if pruneErr != nil {
			m.err = fmt.Errorf("context pruning failed: %w", pruneErr)
		} else if pruned {
			m.err = fmt.Errorf("conversation summarized to fit context limit")
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
		m.Session.Save()

		// Re-trigger Oracle query to evaluate updated facts
		m.loading = true
		m.err = nil
		return m, m.queryOracleCmd("Requirements updated manually via editor.")

	case genProgressMsg:
		m.genStatus = string(msg)
		return m, m.recvGenProgressCmd()

	case genFinishedMsg:
		m.isGenerating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.genStatus = "All specifications synthesized successfully!"
			m.Session.Save() // Save final state
		}
		return m, nil
	}

	// Update text input
	if !m.isCompleted && !m.loading {
		m.textInput, cmd = m.textInput.Update(msg)
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
