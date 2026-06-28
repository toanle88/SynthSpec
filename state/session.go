package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toanle/synthspec/gateway"
)

// GeneratedFileState represents the status and compliance audit of a generated file
type GeneratedFileState struct {
	FileName       string                     `json:"file_name"`
	Results        []gateway.ComplianceResult `json:"results"`
	HasError       bool                       `json:"has_error"`
	ErrorStr       string                     `json:"error_str,omitempty"`
	InProgressText string                     `json:"in_progress_text,omitempty"`
	CurrentAttempt int                        `json:"current_attempt,omitempty"`
}

// Session represents a project session state
type Session struct {
	ProjectName     string                      `json:"project_name"`
	Provider        string                      `json:"provider"`
	Model           string                      `json:"model"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
	History         []gateway.Message           `json:"history"`
	Facts           gateway.Facts               `json:"facts"`
	Scores          gateway.ConfidenceScores    `json:"scores"`
	Rationales      gateway.DimensionRationales `json:"rationales"`
	LastQuestion    string                      `json:"last_question"`
	LastChoices     []string                    `json:"last_choices"`
	TotalTokensUsed int                         `json:"total_tokens_used"`
	GeneratedFiles  []GeneratedFileState        `json:"generated_files,omitempty"`
}

// GetSessionDir returns the project directory path
func GetSessionDir(projectName string) string {
	return filepath.Join("synthspec", projectName)
}

// GetSessionPath returns the project session.json file path
func GetSessionPath(projectName string) string {
	return filepath.Join(GetSessionDir(projectName), "session.json")
}

// Save persists the session state to disk
func (s *Session) Save() error {
	s.UpdatedAt = time.Now()
	dir := GetSessionDir(s.ProjectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	path := GetSessionPath(s.ProjectName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession reads a session state from disk
func LoadSession(projectName string) (*Session, error) {
	path := GetSessionPath(projectName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &s, nil
}

// ListProjects scans the local directory for active SynthSpec projects
func ListProjects() ([]string, error) {
	entries, err := os.ReadDir("synthspec")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			sessionPath := filepath.Join("synthspec", entry.Name(), "session.json")
			if _, err := os.Stat(sessionPath); err == nil {
				projects = append(projects, entry.Name())
			}
		}
	}
	return projects, nil
}

// Model Limits (context window size in tokens)
var modelLimits = map[string]int{
	"gemini-2.5-pro":     2000000,
	"gemini-1.5-pro":     2000000,
	"gemini-1.5-flash":   1000000,
	"gpt-4o":             128000,
	"o3-mini":            200000,
	"claude-3-5-sonnet":  200000,
	"mock-model":         10000,
}

// CheckAndPruneContext evaluates total tokens and runs summarization if over 75% capacity
func (s *Session) CheckAndPruneContext(ctx context.Context, gw gateway.Gateway) (bool, error) {
	limit, exists := modelLimits[s.Model]
	if !exists {
		// Default conservative limit
		limit = 100000
	}

	threshold := int(float64(limit) * 0.75)
	if s.TotalTokensUsed <= threshold {
		return false, nil
	}

	// Summarize conversation history
	summaryPrompt := "Summarize the key architectural choices, user preferences, and engineering requirements established in this chat history. Compress it into a clear, single paragraph summarizing the consensus."
	
	// Create a temporary history for summarization
	sumHistory := append(s.History, gateway.Message{Role: "user", Content: summaryPrompt})
	
	resp, err := gw.QueryOracle(ctx, s.Facts, sumHistory, "")
	if err != nil {
		return false, fmt.Errorf("summarization call failed: %w", err)
	}

	// Reset conversation history to a single condensed context block
	summaryText := "Summary of earlier conversation:\n" + resp.NextQuestion // Using next_question as the return channel in standard QueryOracle
	if summaryText == "" {
		summaryText = "Summarized historical progress."
	}

	s.History = []gateway.Message{
		{Role: "user", Content: "Let's summarize our progress so far."},
		{Role: "assistant", Content: summaryText},
	}
	s.TotalTokensUsed = resp.TokensPrompt + resp.TokensCompletion

	if err := s.Save(); err != nil {
		return true, fmt.Errorf("failed to save session after pruning: %w", err)
	}

	return true, nil
}

// AddTurn appends a conversation turn and updates the total tokens
func (s *Session) AddTurn(userMsg, assistantMsg string, tokensPrompt, tokensCompletion int) {
	s.History = append(s.History, gateway.Message{Role: "user", Content: userMsg})
	s.History = append(s.History, gateway.Message{Role: "assistant", Content: assistantMsg})
	s.TotalTokensUsed = tokensPrompt + tokensCompletion
}

// LogError writes an error message with a timestamp to the project's error log file.
// If projectName is empty, it writes to the global "synthspec/errors.log" file.
func LogError(projectName string, err error) {
	if err == nil {
		return
	}

	// Create "synthspec" root directory if it doesn't exist
	if errMk := os.MkdirAll("synthspec", 0755); errMk != nil {
		return // Silently fail if we can't write to disk
	}

	var logPath string
	if projectName != "" {
		projDir := GetSessionDir(projectName)
		if errMk := os.MkdirAll(projDir, 0755); errMk == nil {
			logPath = filepath.Join(projDir, "errors.log")
		} else {
			logPath = filepath.Join("synthspec", "errors.log")
		}
	} else {
		logPath = filepath.Join("synthspec", "errors.log")
	}

	f, errOpen := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if errOpen != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] ERROR: %v\n", timestamp, err)
	_, _ = f.WriteString(logEntry)
}

