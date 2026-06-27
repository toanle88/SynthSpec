package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

// TelemetryMetadata represents the .synthspec-meta.json structure
type TelemetryMetadata struct {
	ProjectName         string            `json:"project_name"`
	GenerationTimestamp string            `json:"generation_timestamp"`
	EngineVersion       string            `json:"engine_version"`
	ProviderUsed        string            `json:"provider_used"`
	CompletionMetrics   CompletionMetrics `json:"completion_metrics"`
}

type CompletionMetrics struct {
	TotalTurns     int `json:"total_turns"`
	TokensConsumed int `json:"tokens_consumed"`
}

// Backlog represents the top-level structure of the engineering backlog
type Backlog struct {
	Epics []Epic `json:"epics"`
}

// Epic represents a high-level feature category containing tasks
type Epic struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tasks       []Task `json:"tasks"`
}

// Task represents a development task in the backlog
type Task struct {
	ID                 string   `json:"id"`
	Summary            string   `json:"summary"`
	Details            string   `json:"details"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// sanitizeJSONOutput strips markdown code block fences if they exist
func sanitizeJSONOutput(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		// Find first newline
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
		if strings.HasSuffix(content, "```") {
			content = content[:len(content)-3]
		}
		content = strings.TrimSpace(content)
	}
	return content
}

// validateBacklog parses and validates the engineering backlog JSON against structural requirements
func validateBacklog(content string) error {
	var backlog Backlog
	if err := json.Unmarshal([]byte(content), &backlog); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	if len(backlog.Epics) == 0 {
		return fmt.Errorf("backlog must contain at least one epic")
	}

	for i, epic := range backlog.Epics {
		if epic.ID == "" {
			return fmt.Errorf("epic %d is missing ID", i)
		}
		if epic.Title == "" {
			return fmt.Errorf("epic %s is missing Title", epic.ID)
		}
		if epic.Description == "" {
			return fmt.Errorf("epic %s is missing Description", epic.ID)
		}
		if len(epic.Tasks) == 0 {
			return fmt.Errorf("epic %s must contain at least one task", epic.ID)
		}
		for j, task := range epic.Tasks {
			if task.ID == "" {
				return fmt.Errorf("task %d in epic %s is missing ID", j, epic.ID)
			}
			if task.Summary == "" {
				return fmt.Errorf("task %s in epic %s is missing Summary", task.ID, epic.ID)
			}
			if task.Details == "" {
				return fmt.Errorf("task %s in epic %s is missing Details", task.ID, epic.ID)
			}
			if len(task.AcceptanceCriteria) == 0 {
				return fmt.Errorf("task %s in epic %s must contain at least one acceptance criterion", task.ID, epic.ID)
			}
		}
	}
	return nil
}

// Generate runs sequential spec generation for all files
func Generate(ctx context.Context, gw gateway.Gateway, sess *state.Session, outputDir string, progress chan<- string) error {
	defer close(progress)

	// Ensure output directory exists
	if outputDir == "" {
		outputDir = filepath.Join(state.GetSessionDir(sess.ProjectName), "output")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	files := []string{
		"01_prd_functional.md",
		"02_system_architecture.md",
		"03_security_threat_model.md",
		"04_openapi_contract.yaml",
		"05_engineering_backlog.json",
	}

	for _, fileName := range files {
		progress <- fmt.Sprintf("Synthesizing %s...", fileName)

		var content string
		var err error
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			content, err = gw.GenerateSpecFile(ctx, sess.Facts, fileName)
			if err != nil {
				if attempt == maxRetries {
					return fmt.Errorf("failed to generate %s after %d attempts: %w", fileName, maxRetries, err)
				}
				progress <- fmt.Sprintf("Error generating %s (attempt %d/%d): %v. Retrying...", fileName, attempt, maxRetries, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Perform validation for specific files (e.g. JSON backlog)
			if fileName == "05_engineering_backlog.json" {
				sanitized := sanitizeJSONOutput(content)
				if valErr := validateBacklog(sanitized); valErr != nil {
					err = valErr
					if attempt == maxRetries {
						return fmt.Errorf("failed to validate %s after %d attempts: %w. Content: %s", fileName, maxRetries, valErr, content)
					}
					progress <- fmt.Sprintf("Validation failed for %s (attempt %d/%d): %v. Retrying...", fileName, attempt, maxRetries, valErr)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				content = sanitized
			}

			err = nil
			break
		}

		filePath := filepath.Join(outputDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s output file: %w", fileName, err)
		}
	}

	// Generate .synthspec-meta.json
	progress <- "Compiling solution metadata (.synthspec-meta.json)..."

	meta := TelemetryMetadata{
		ProjectName:         sess.ProjectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       "1.0.0",
		ProviderUsed:        sess.Provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(sess.History) / 2, // Pairs of user/assistant turns
			TokensConsumed: sess.TotalTokensUsed,
		},
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize telemetry metadata: %w", err)
	}

	metaPath := filepath.Join(outputDir, ".synthspec-meta.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write .synthspec-meta.json: %w", err)
	}

	progress <- fmt.Sprintf("All files generated in: %s", outputDir)
	return nil
}

