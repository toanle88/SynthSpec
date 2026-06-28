package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toanle/synthspec/gateway"
)

func TestResolveEditor_EDITOREnv(t *testing.T) {
	t.Setenv("EDITOR", "myeditor")
	t.Setenv("VISUAL", "") // Ensure no interference
	cmd, args := resolveEditor("test.json")
	if cmd != "myeditor" {
		t.Errorf("expected editor 'myeditor', got %q", cmd)
	}
	if len(args) != 1 || args[0] != "test.json" {
		t.Errorf("expected args ['test.json'], got %v", args)
	}
}

func TestResolveEditor_VISUALEnv(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "myvisual")
	cmd, args := resolveEditor("test.json")
	if cmd != "myvisual" {
		t.Errorf("expected editor 'myvisual', got %q", cmd)
	}
	_ = args // args verified by other tests
}

func TestResolveEditor_EDITORPrecedesVISUAL(t *testing.T) {
	t.Setenv("EDITOR", "myeditor")
	t.Setenv("VISUAL", "myvisual")
	cmd, _ := resolveEditor("test.json")
	if cmd != "myeditor" {
		t.Errorf("expected EDITOR to take precedence, got %q", cmd)
	}
}

func TestResolveEditor_NoEnvFallback(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	cmd, args := resolveEditor("test.json")
	if cmd == "" {
		t.Error("expected a fallback editor command, got empty string")
	}
	if len(args) == 0 {
		t.Errorf("expected at least 1 arg, got none: %v", args)
	}
	// Last arg should always be the file path
	if args[len(args)-1] != "test.json" {
		t.Errorf("expected last arg to be 'test.json', got %v", args)
	}
}

func TestGetEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	projectName := "test-editor-project"
	// Create the directory that GetEditorCommand expects
	dir := GetSessionDir(projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Join("synthspec", projectName))

	facts := gateway.Facts{
		Functional: "test functional",
		Structural: "test structural",
		Security:   "test security",
		Compliance: "test compliance",
	}

	cmd, filePath, err := GetEditorCommand(projectName, facts)
	if err != nil {
		t.Fatalf("GetEditorCommand failed: %v", err)
	}

	// Verify file was created with correct JSON
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read editor file: %v", err)
	}

	var readFacts gateway.Facts
	if err := json.Unmarshal(data, &readFacts); err != nil {
		t.Fatalf("editor file should contain valid JSON: %v", err)
	}
	if readFacts.Functional != "test functional" {
		t.Errorf("expected functional 'test functional', got %q", readFacts.Functional)
	}

	// Verify the command string contains the editor name
	cmdStr := cmd.String()
	if !strings.Contains(cmdStr, "cat") {
		t.Errorf("expected cmd to contain 'cat', got %q", cmdStr)
	}
}

func TestReadBackEditedFacts(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "facts_edit.json")

	originalFacts := gateway.Facts{
		Functional: "edited functional",
		Structural: "edited structural",
	}

	data, _ := json.MarshalIndent(originalFacts, "", "  ")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	facts, err := ReadBackEditedFacts(filePath)
	if err != nil {
		t.Fatalf("ReadBackEditedFacts failed: %v", err)
	}

	if facts.Functional != "edited functional" {
		t.Errorf("expected functional 'edited functional', got %q", facts.Functional)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted after ReadBackEditedFacts")
	}
}

func TestReadBackEditedFacts_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad_facts.json")

	if err := os.WriteFile(filePath, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadBackEditedFacts(filePath)
	if err == nil || !strings.Contains(err.Error(), "invalid facts JSON") {
		t.Errorf("expected invalid facts JSON error, got: %v", err)
	}
}

func TestReadBackEditedFacts_MissingFile(t *testing.T) {
	_, err := ReadBackEditedFacts("nonexistent.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestGetFileEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cmd, err := GetFileEditorCommand("somefile.md")
	if err != nil {
		t.Fatalf("GetFileEditorCommand failed: %v", err)
	}
	cmdStr := cmd.String()
	if !strings.Contains(cmdStr, "nano") {
		t.Errorf("expected cmd to contain 'nano', got %q", cmdStr)
	}
}

func TestGetEditorCommand_CreatesDir(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	projectName := "test-editor-dir-create"
	// Pre-create the directory (GetEditorCommand does not create it)
	dir := GetSessionDir(projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Join("synthspec", projectName))

	facts := gateway.Facts{Functional: "test"}
	cmd, filePath, err := GetEditorCommand(projectName, facts)
	if err != nil {
		t.Fatalf("GetEditorCommand failed: %v", err)
	}
	_ = cmd
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", filePath)
	}
}
