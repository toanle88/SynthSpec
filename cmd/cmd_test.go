package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui"
)

func TestMain(m *testing.M) {
	// Stub out the TUI execution so we don't start the Bubble Tea terminal interface
	runTUI = func(m tui.DashboardModel) error {
		return nil
	}
	os.Exit(m.Run())
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
}
