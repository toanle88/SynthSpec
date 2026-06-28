package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/toanle/synthspec/gateway"
)

func TestSessionSaveAndLoad(t *testing.T) {
	projectName := "test-session-project"
	defer os.RemoveAll(filepath.Join("synthspec", projectName)) // Clean up

	sess := Session{
		ProjectName: projectName,
		Provider:    "mock",
		Model:       "mock-model",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Facts: gateway.Facts{
			Functional: "functional requirements",
			Structural: "structural requirements",
		},
		Scores: gateway.ConfidenceScores{
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
	
	defer os.RemoveAll(filepath.Join("synthspec", project1))
	defer os.RemoveAll(filepath.Join("synthspec", project2))

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
	defer os.RemoveAll(filepath.Join("synthspec", projectName))
	defer os.Remove(filepath.Join("synthspec", "errors.log"))

	// Test project-specific error logging
	errSample := fmt.Errorf("sample error description")
	LogError(projectName, errSample)

	logPath := filepath.Join("synthspec", projectName, "errors.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read project log file: %v", err)
	}

	if !strings.Contains(string(content), "ERROR: sample error description") {
		t.Errorf("expected log file to contain error description, got: %s", string(content))
	}

	// Test global error logging
	errGlobalSample := fmt.Errorf("global sample error description")
	LogError("", errGlobalSample)

	globalLogPath := filepath.Join("synthspec", "errors.log")
	globalContent, err := os.ReadFile(globalLogPath)
	if err != nil {
		t.Fatalf("failed to read global log file: %v", err)
	}

	if !strings.Contains(string(globalContent), "ERROR: global sample error description") {
		t.Errorf("expected global log file to contain error description, got: %s", string(globalContent))
	}
}

