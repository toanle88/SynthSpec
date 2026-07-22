package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/logger"
)

func (m DashboardModel) recvThoughtCmd() tea.Cmd {
	return func() tea.Msg {
		token, ok := <-m.thoughtChan
		if !ok {
			return streamDoneMsg{}
		}

		// Wait briefly to accumulate multiple incoming tokens that arrive in close succession
		time.Sleep(50 * time.Millisecond)

		var batch strings.Builder
		batch.WriteString(token)

		// Drain any other buffered tokens currently waiting in the channel (non-blocking)
		for {
			select {
			case t, open := <-m.thoughtChan:
				if !open {
					return thoughtTokenMsg(batch.String())
				}
				batch.WriteString(t)
			default:
				return thoughtTokenMsg(batch.String())
			}
		}
	}
}

// Background commands
// queryOracleCmd submits requirement definitions asynchronously to the LLM Oracle model.
func (m DashboardModel) queryOracleCmd(latestInput string) tea.Cmd {
	logger.LogEvent("TUI", fmt.Sprintf("Querying Oracle with latestInput (length: %d)", len(latestInput)))
	return func() tea.Msg {
		timeoutSec := 300
		if m.Settings != nil && m.Settings.TimeoutSeconds > 0 {
			timeoutSec = m.Settings.TimeoutSeconds
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
		defer cancel()

		// If user answer was provided, append it to history beforehand
		if latestInput != "" && latestInput != manualUpdateMsg {
			m.Session.AddTurn(latestInput, m.Session.GetLastQuestion(), 0, 0)
		}

		resp, err := m.Gateway.QueryOracleStream(ctx, m.Session.GetFacts(), m.Session.GetHistory(), latestInput, m.Session.GetScores(), m.Session.GetRationales(), m.thoughtChan)
		if err != nil {
			return oracleResultMsg{err: err}
		}

		// Update tokens in session (will be saved in Update msg handler)
		if latestInput != "" && latestInput != manualUpdateMsg {
			// Back-fill actual assistant response
			m.Session.GetHistory()[len(m.Session.GetHistory())-1].Content = resp.NextQuestion
		}
		_ = m.Session.UpdateTokens(resp.TokensPrompt, resp.TokensCompletion)

		return oracleResultMsg{resp: resp}
	}
}

// Receives generator logs asynchronously
// recvGenProgressCmd reads progress logs asynchronously from the pipeline worker channel.
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
// generateSpecsCmd synthesizes all targets in parallel inside background worker goroutines.
func (m DashboardModel) generateSpecsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		err := generator.Generate(ctx, m.Gateway, m.Session, m.OutputDir, m.genChan, m.approvalChan, m.diffApprovalChan, m.forceFinishChan)
		return genFinishedMsg{err: err}
	}
}

// pruneContextCmd triggers context history summarization when tokens limit is exceeded.
func (m DashboardModel) pruneContextCmd() tea.Cmd {
	return func() tea.Msg {
		pruned, err := m.Session.CheckAndPruneContext(context.Background(), m.Gateway)
		return contextPruneResultMsg{pruned: pruned, err: err}
	}
}
