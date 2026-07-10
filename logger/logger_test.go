package logger

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerInitializationAndLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthspec-logger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("SYNTHSPEC_ROOT", tempDir)

	// Setup test environment
	logDir := filepath.Join(tempDir, ".synthspec")
	logPath := filepath.Join(logDir, "crash.log")

	// Clean up any existing logs
	_ = os.Remove(logPath)

	// 1. Check disabled state
	err = Init(false, false)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	Log("Should not be written")
	if _, err := os.Stat(logPath); err == nil {
		t.Error("log file was created when logging is disabled")
	}

	// 2. Check enabled state (via CLI debug)
	err = Init(true, false)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	Log("Hello debug logger: %s", "test-arg")
	LogAPI("test-provider", "test-model", 123*time.Millisecond, 10, 20, nil)
	LogAPI("test-provider", "test-model", 456*time.Millisecond, 0, 0, errors.New("test API error"))

	Close()

	// 3. Verify content
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read log file contents: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "SynthSpec Session Started") {
		t.Error("missing startup marker")
	}
	if !strings.Contains(content, "Hello debug logger: test-arg") {
		t.Error("missing debug log entry")
	}
	if !strings.Contains(content, "Provider: test-provider | Model: test-model | Status: SUCCESS | Duration: 123ms | Tokens: Prompt=10, Completion=20") {
		t.Error("missing successful API log entry")
	}
	if !strings.Contains(content, "Status: ERROR | Duration: 456ms | Tokens: Prompt=0, Completion=0 | Error: test API error") {
		t.Error("missing error API log entry")
	}

	// Clean up after test
	Close()
	_ = os.Remove(logPath)
}
