package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const timeFormat = "2006-01-02 15:04:05.000"

var (
	enabled bool
	mu      sync.Mutex
	logFile *os.File
)

// Init initializes the logging system. It enables logging if either the cli flag is true or settings show debug is true.
func Init(cliDebug, settingsDebug bool) error {
	mu.Lock()
	defer mu.Unlock()

	// Close existing log file if open
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	enabled = cliDebug || settingsDebug
	if !enabled {
		return nil
	}

	dir := ".synthspec"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(dir, "crash.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logFile = file

	// Write a startup marker
	timestamp := time.Now().Format(timeFormat)
	fmt.Fprintf(logFile, "\n[%s] [SYSTEM] --- SynthSpec Session Started (CLI Debug: %t, Settings Debug: %t) ---\n", timestamp, cliDebug, settingsDebug)
	return nil
}

// Close closes the log file if it's open.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// Log writes a generic message to the log file if debugging is enabled.
func Log(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format(timeFormat)
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] [DEBUG] %s\n", timestamp, msg)
}

// LogEvent writes a structured component event trace to the log file.
func LogEvent(component, event string) {
	Log("[%s] %s", component, event)
}

// LogAPI logs sanitized API request/response metadata without sensitive content.
func LogAPI(provider, model string, duration time.Duration, promptTokens, completionTokens int, err error) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format(timeFormat)
	status := "SUCCESS"
	errStr := ""
	if err != nil {
		status = "ERROR"
		errStr = fmt.Sprintf(" | Error: %v", err)
	}

	fmt.Fprintf(logFile, "[%s] [API] Provider: %s | Model: %s | Status: %s | Duration: %s | Tokens: Prompt=%d, Completion=%d%s\n",
		timestamp, provider, model, status, duration.String(), promptTokens, completionTokens, errStr)
}
