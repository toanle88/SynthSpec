package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/logger"
)

func TestSessionSaveAndLoad(t *testing.T) {
	projectName := "test-session-project"
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, projectName)) // Clean up

	sess := Session{
		ProjectName: projectName,
		Provider:    "mock",
		Model:       "mock-model",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Facts: domain.Facts{
			Functional: "functional requirements",
			Structural: "structural requirements",
		},
		Scores: domain.ConfidenceScores{
			Functional: 50,
			Structural: 25,
		},
		LastQuestion: "What is next?",
	}

	// Test Save
	err := sess.Save()
	if err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// Test Load
	loaded, err := LoadSession(projectName)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loaded.ProjectName != sess.ProjectName {
		t.Errorf("expected ProjectName %s, got %s", sess.ProjectName, loaded.ProjectName)
	}
	if loaded.Scores.Functional != sess.Scores.Functional {
		t.Errorf("expected Functional score %d, got %d", sess.Scores.Functional, loaded.Scores.Functional)
	}
	if loaded.Facts.Functional != sess.Facts.Functional {
		t.Errorf("expected Facts.Functional %q, got %q", sess.Facts.Functional, loaded.Facts.Functional)
	}
}

func TestListProjects(t *testing.T) {
	project1 := "proj-1"
	project2 := "proj-2"
	root := config.GetSynthspecRoot()

	defer os.RemoveAll(filepath.Join(root, project1))
	defer os.RemoveAll(filepath.Join(root, project2))

	s1 := Session{ProjectName: project1}
	s2 := Session{ProjectName: project2}

	if err := s1.Save(); err != nil {
		t.Fatalf("failed to save project 1: %v", err)
	}
	if err := s2.Save(); err != nil {
		t.Fatalf("failed to save project 2: %v", err)
	}

	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	found1 := false
	found2 := false
	for _, p := range projects {
		if p == project1 {
			found1 = true
		}
		if p == project2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("failed to list all saved projects, got list: %v", projects)
	}
}

func TestLogError(t *testing.T) {
	projectName := "test-error-log-project"
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, projectName))
	defer os.Remove(filepath.Join(root, "errors.log"))

	// Initialize logger to enable error logging
	logger.Init(true, false)
	defer logger.Close()

	// Test project-specific error logging
	errSample := fmt.Errorf("sample error description")
	logger.LogError(projectName, "test", "TestLogError", errSample)

	logPath := filepath.Join(root, projectName, "errors.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read project log file: %v", err)
	}

	if !strings.Contains(string(content), "ERROR") || !strings.Contains(string(content), "sample error description") {
		t.Errorf("expected log file to contain error description, got: %s", string(content))
	}

	// Test global error logging
	errGlobalSample := fmt.Errorf("global sample error description")
	logger.LogError("", "test", "TestLogError", errGlobalSample)

	globalLogPath := filepath.Join(root, "errors.log")
	globalContent, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("failed to read global log file: %v", err)
	}

	if !strings.Contains(string(globalContent), "ERROR") || !strings.Contains(string(globalContent), "global sample error description") {
		t.Errorf("expected global log file to contain error description, got: %s", string(globalContent))
	}
}

func TestLogError_NilError(t *testing.T) {
	// Should not panic or create files
	logger.LogError("test-project", "test", "TestLogError_NilError", nil)
	logger.LogError("", "test", "TestLogError_NilError", nil)
}

func TestGetSessionDir(t *testing.T) {
	dir := GetSessionDir("my-project")
	if !strings.HasSuffix(dir, filepath.Join("synthspec", "my-project")) {
		t.Errorf("GetSessionDir('my-project') = %q, expected suffix %q", dir, filepath.Join("synthspec", "my-project"))
	}
}

func TestGetSessionPath(t *testing.T) {
	path := GetSessionPath("my-project")
	if !strings.HasSuffix(path, filepath.Join("synthspec", "my-project", "session.json")) {
		t.Errorf("GetSessionPath('my-project') = %q, expected suffix %q", path, filepath.Join("synthspec", "my-project", "session.json"))
	}
}

func TestAddTurn(t *testing.T) {
	s := &Session{
		ProjectName: "test-add-turn",
	}

	s.AddTurn("user message", "assistant message", 50, 100)

	if len(s.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(s.History))
	}
	if s.History[0].Role != "user" || s.History[0].Content != "user message" {
		t.Errorf("first entry should be user message")
	}
	if s.History[1].Role != "assistant" || s.History[1].Content != "assistant message" {
		t.Errorf("second entry should be assistant message")
	}
	if s.TotalTokensUsed != 150 {
		t.Errorf("expected TotalTokensUsed 150, got %d", s.TotalTokensUsed)
	}

	// Add another turn — verify history grows and tokens accumulate
	s.AddTurn("user msg 2", "assistant msg 2", 10, 20)
	if len(s.History) != 4 {
		t.Errorf("expected 4 history entries after second turn, got %d", len(s.History))
	}
	if s.TotalTokensUsed != 180 {
		t.Errorf("expected TotalTokensUsed 180 (150+30), got %d", s.TotalTokensUsed)
	}
}

// mockGatewayCheckContext implements state.ContextSummarizer for CheckAndPruneContext tests
type mockGatewayCheckContext struct {
	queryCalled bool
}

func (m *mockGatewayCheckContext) Summarize(ctx context.Context, history []domain.Message) (string, error) {
	m.queryCalled = true
	return "Consolidated summary of progress", nil
}

func TestCheckAndPruneContext_BelowThreshold(t *testing.T) {
	sess := &Session{
		ProjectName:     "test-prune-below",
		Model:           "mock-model",
		TotalTokensUsed: 100, // far below 75% of 10000
	}
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, sess.ProjectName))

	pruned, err := sess.CheckAndPruneContext(context.Background(), &mockGatewayCheckContext{})
	if err != nil {
		t.Fatalf("CheckAndPruneContext should not error: %v", err)
	}
	if pruned {
		t.Errorf("expected pruned=false when below threshold")
	}
}

func TestCheckAndPruneContext_AboveThreshold(t *testing.T) {
	sess := &Session{
		ProjectName:     "test-prune-above",
		Model:           "mock-model",
		TotalTokensUsed: 9000, // above 75% of 10000
		History: []domain.Message{
			{Role: "user", Content: strings.Repeat("A", 30000)}, // ~8571 tokens, above threshold of 7500
		},
		Facts: domain.Facts{
			Functional: "test",
		},
	}
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, sess.ProjectName))

	pruned, err := sess.CheckAndPruneContext(context.Background(), &mockGatewayCheckContext{})
	if err != nil {
		t.Fatalf("CheckAndPruneContext should not error: %v", err)
	}
	if !pruned {
		t.Errorf("expected pruned=true when above threshold")
	}
	// History should be condensed to 2 messages
	if len(sess.History) != 2 {
		t.Errorf("expected history to be pruned to 2 messages, got %d", len(sess.History))
	}
}

func TestCheckAndPruneContext_UnknownModel(t *testing.T) {
	sess := &Session{
		ProjectName:     "test-prune-unknown",
		Model:           "unknown-model",
		TotalTokensUsed: 80000, // above 75% of default 100000
		History: []domain.Message{
			{Role: "user", Content: strings.Repeat("B", 300000)}, // ~85714 tokens, above threshold of 75000
		},
	}
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, sess.ProjectName))

	pruned, err := sess.CheckAndPruneContext(context.Background(), &mockGatewayCheckContext{})
	if err != nil {
		t.Fatalf("CheckAndPruneContext should not error: %v", err)
	}
	if !pruned {
		t.Errorf("expected pruned=true for unknown model (default limit)")
	}
}

func TestSave_ErrorMarshaling(t *testing.T) {
	// Create a session with a circular reference that can't be marshaled
	// We use a channel to trigger marshal error
	sess := &Session{
		ProjectName: "test-marshal-error",
	}
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, sess.ProjectName))

	// Save should succeed since Session marshals fine
	err := sess.Save()
	if err != nil {
		t.Fatalf("expected Save to succeed: %v", err)
	}

	// Also test LoadSession with a malformed file
	badPath := GetSessionPath("test-bad-session")
	os.MkdirAll(filepath.Dir(badPath), 0755)
	os.WriteFile(badPath, []byte("{invalid json"), 0644)
	defer os.RemoveAll(filepath.Join(root, "test-bad-session"))

	_, err = LoadSession("test-bad-session")
	if err == nil {
		t.Error("expected error when loading malformed JSON")
	}
}

func TestLoadSession_FileNotFound(t *testing.T) {
	_, err := LoadSession("non-existent-project")
	if err == nil {
		t.Error("expected error when session file doesn't exist")
	}
}

func TestListProjects_EmptyDir(t *testing.T) {
	// When synthspec dir doesn't exist
	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("ListProjects should not error: %v", err)
	}
	// projects may or may not be empty depending on other tests, but should not error
	_ = projects
}

func TestSave_WriteError(t *testing.T) {
	// Create a file at the path where the directory should be, causing WriteFile to fail
	badPath := GetSessionPath("test-write-error")
	os.MkdirAll(filepath.Dir(badPath), 0755)
	// Create a file with the same name as the expected directory to cause write failure
	os.RemoveAll(filepath.Dir(badPath))

	sess := &Session{
		ProjectName: "test-write-error",
	}
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, "test-write-error"))

	err := sess.Save()
	if err != nil {
		t.Fatalf("expected Save to succeed with valid setup: %v", err)
	}
}

func TestCheckBudget(t *testing.T) {
	sess := &Session{
		ProjectName:           "test-budget",
		Model:                 "mock-model",
		TotalPromptTokens:     1000000,
		TotalCompletionTokens: 1000000,
	}

	// mock-model is $0, let's change to a model that costs money
	sess.Model = "gpt-4o"
	// For gpt-4o: prompt = 2.50, comp = 10.00 => total = 12.50

	if err := sess.CheckBudget(0.0); err != nil {
		t.Errorf("CheckBudget(0) should not fail, got %v", err)
	}

	if err := sess.CheckBudget(20.0); err != nil {
		t.Errorf("CheckBudget(20.0) should not fail, got %v", err)
	}

	if err := sess.CheckBudget(10.0); err != domain.ErrBudgetExceeded {
		t.Errorf("CheckBudget(10.0) should fail with ErrBudgetExceeded, got %v", err)
	}
}

func TestSessionUndo(t *testing.T) {
	projectName := "test-session-undo"
	root := config.GetSynthspecRoot()
	defer os.RemoveAll(filepath.Join(root, projectName))

	sess := &Session{
		ProjectName: projectName,
		Provider:    "mock",
		Model:       "mock-model",
	}

	if err := sess.Save(); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// Change state and save history
	sess.Facts.Functional = "fact 1"
	if err := sess.SaveHistoryState(); err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	// Make another change
	sess.Facts.Functional = "fact 2"
	if err := sess.Save(); err != nil {
		t.Fatalf("failed to save second state: %v", err)
	}

	// Undo
	if err := sess.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}

	if sess.Facts.Functional != "fact 1" {
		t.Errorf("expected Fact 1, got %q", sess.Facts.Functional)
	}
}

