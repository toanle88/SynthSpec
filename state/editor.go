package state

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/toanle/synthspec/domain"
)

// GetEditorCommand prepares the temp file and returns the exec.Cmd to be run by Bubble Tea.
func GetEditorCommand(projectName string, facts domain.Facts) (*exec.Cmd, string, error) {
	dir := GetSessionDir(projectName)
	filePath := filepath.Join(dir, "facts_edit.json")

	data, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize facts: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, "", fmt.Errorf("failed to write facts edit file: %w", err)
	}

	editorCmd, editorArgs := resolveEditor(filePath)
	return exec.Command(editorCmd, editorArgs...), filePath, nil
}

// ReadBackEditedFacts parses the edited file back and removes it.
func ReadBackEditedFacts(filePath string) (domain.Facts, error) {
	defer os.Remove(filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return domain.Facts{}, fmt.Errorf("failed to read back facts: %w", err)
	}

	var facts domain.Facts
	if err := json.Unmarshal(data, &facts); err != nil {
		return domain.Facts{}, fmt.Errorf("invalid facts JSON: %w", err)
	}

	return facts, nil
}

func resolveEditor(filePath string) (string, []string) {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, []string{filePath}
	}
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual, []string{filePath}
	}

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("code"); err == nil {
			return "code", []string{"--wait", filePath}
		}
		return "notepad.exe", []string{filePath}
	}

	for _, fallback := range []string{"nano", "vim", "vi"} {
		if _, err := exec.LookPath(fallback); err == nil {
			return fallback, []string{filePath}
		}
	}

	return "vi", []string{filePath}
}

// GetFileEditorCommand returns the exec.Cmd to edit an arbitrary file using the resolved system editor.
func GetFileEditorCommand(filePath string) (*exec.Cmd, error) {
	editorCmd, editorArgs := resolveEditor(filePath)
	return exec.Command(editorCmd, editorArgs...), nil
}
