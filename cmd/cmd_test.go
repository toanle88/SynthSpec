package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui"
)

func TestMain(m *testing.M) {
	// Stub out the TUI execution so we don't start the Bubble Tea terminal interface
	runTUI = func(_ tui.DashboardModel) error {
		return nil
	}
	os.Exit(m.Run())
}

// cleanupProject removes a project from the session directory to ensure test isolation
func cleanupProject(name string) {
	path := state.GetSessionDir(name)
	os.RemoveAll(path)
}

func TestInitAndResumeCmd(t *testing.T) {
	// Create a temp directory for session isolation
	tempDir, err := os.MkdirTemp("", "synthspec-cmd-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change current working directory to tempDir so that session files are created there
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// 1. Test "init" command with mock gateway
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Set CLI arguments and flags
	mockFlag = true
	rootCmd.SetArgs([]string{"init", "my-test-proj"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init cmd: %v", err)
	}

	// Verify project session directory and session.json file were created
	sessionPath := state.GetSessionPath("my-test-proj")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("expected session.json to be created at %s, but got error: %v", sessionPath, err)
	}

	// 2. Test "init" of an existing project fails
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when initializing an already existing project, got nil")
	}

	// 3. Test "resume" command
	rootCmd.SetArgs([]string{"resume", "my-test-proj"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute resume cmd: %v", err)
	}

	// 4. Test "resume" with invalid project name fails
	rootCmd.SetArgs([]string{"resume", "non-existent-proj"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when resuming non-existent project, got nil")
	}

	// 5. Test "update" command
	rootCmd.SetArgs([]string{"update", "my-test-proj"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute update cmd: %v", err)
	}

	// 6. Test "update" with invalid project name fails
	rootCmd.SetArgs([]string{"update", "non-existent-proj"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when updating non-existent project, got nil")
	}

	// 7. Test "list" command
	rootCmd.SetArgs([]string{"list"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute list cmd: %v", err)
	}

	// 8. Test "delete" command with invalid project fails
	rootCmd.SetArgs([]string{"delete", "non-existent-proj"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when deleting non-existent project, got nil")
	}

	// 9. Test "delete" command with --force flag
	forceFlag = true
	rootCmd.SetArgs([]string{"delete", "my-test-proj"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute delete cmd: %v", err)
	}

	// Verify project directory is gone
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatalf("expected project directory to be deleted, but it still exists at %s", sessionPath)
	}
}

func TestInitBlueprint(t *testing.T) {
	// Clean up any leftover state from previous runs
	cleanupProject("bp-proj")

	// Create a temp directory for session isolation
	tempDir, err := os.MkdirTemp("", "synthspec-blueprint-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	mockFlag = true

	// Reset/clear flags
	blueprintFlag = ""

	rootCmd.SetArgs([]string{"init", "bp-proj", "-b", "fintech-saas"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init with blueprint: %v", err)
	}

	sess, err := state.LoadSession("bp-proj")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if sess.Facts.Functional == "" {
		t.Errorf("expected session facts to be pre-populated, but functional facts are empty")
	}

	if !strings.Contains(sess.Facts.Compliance, "PCI-DSS") {
		t.Errorf("expected compliance facts to contain PCI-DSS, got: %s", sess.Facts.Compliance)
	}
}

func TestGetGatewayForSession_Mock(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-gw",
		Provider:    "mock",
		Model:       "mock-model",
	}
	gw, err := NewGatewayForSession(sess, false)
	if err != nil {
		t.Fatalf("expected success with mock provider, got: %v", err)
	}
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
}

func TestGetGatewayForSession_ForceMock(t *testing.T) {
	sess := &state.Session{
		ProjectName: "test-gw-force",
		Provider:    "anthropic",
		Model:       "claude-3-5-sonnet",
	}
	gw, err := NewGatewayForSession(sess, true)
	if err != nil {
		t.Fatalf("expected success with forceMock=true, got: %v", err)
	}
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
}
