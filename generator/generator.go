package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
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
	ComplianceSummary   map[string]int    `json:"compliance_summary,omitempty"`
}

type CompletionMetrics struct {
	TotalTurns     int `json:"total_turns"`
	TokensConsumed int `json:"tokens_consumed"`
}

// ProgressEvent represents a structured progress update sent to the TUI
type ProgressEvent struct {
	File    string `json:"file,omitempty"`
	Status  string `json:"status,omitempty"` // pending, skipped, synthesizing, correcting, auditing, refining, done, failed, started, completed, compiling_report, compiling_metadata
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

func sendProgress(progress chan<- string, event ProgressEvent) {
	if data, err := json.Marshal(event); err == nil {
		progress <- string(data)
	} else {
		progress <- event.Message
	}
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

	// Load quality standards configuration
	standards, err := config.LoadStandards()
	if err != nil {
		return fmt.Errorf("failed to load quality standards: %w", err)
	}

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
		"04_api_architecture_integration.md",
		"05_coding_standards_guidelines.md",
	}

	sendProgress(progress, ProgressEvent{Status: "started", Details: strings.Join(files, ","), Message: "Starting spec generation..."})

	var fileCompliances []FileCompliance

	for _, fileName := range files {
		// Check if file is already successfully generated in session and exists on disk
		cachedIdx := -1
		for idx, gf := range sess.GeneratedFiles {
			if gf.FileName == fileName {
				cachedIdx = idx
				break
			}
		}

		filePath := filepath.Join(outputDir, fileName)
		_, statErr := os.Stat(filePath)

		if cachedIdx != -1 && statErr == nil && !sess.GeneratedFiles[cachedIdx].HasError {
			sendProgress(progress, ProgressEvent{
				File:    fileName,
				Status:  "skipped",
				Details: "already generated",
				Message: fmt.Sprintf("Skipping %s (already generated)", fileName),
			})
			fileCompliances = append(fileCompliances, FileCompliance{
				FileName: fileName,
				Results:  sess.GeneratedFiles[cachedIdx].Results,
				Err:      nil,
			})
			continue
		}

		var content string
		var err error
		maxRetries := 10
		startAttempt := 1
		resuming := false

		if cachedIdx != -1 && sess.GeneratedFiles[cachedIdx].InProgressText != "" {
			content = sess.GeneratedFiles[cachedIdx].InProgressText
			startAttempt = sess.GeneratedFiles[cachedIdx].CurrentAttempt
			if startAttempt < 1 {
				startAttempt = 1
			}
			resuming = true
			sendProgress(progress, ProgressEvent{
				File:    fileName,
				Status:  "synthesizing",
				Details: fmt.Sprintf("resuming from attempt %d/%d", startAttempt, maxRetries),
				Message: fmt.Sprintf("Resuming %s synthesis from attempt %d...", fileName, startAttempt),
			})
		} else {
			sendProgress(progress, ProgressEvent{
				File:    fileName,
				Status:  "synthesizing",
				Details: fmt.Sprintf("attempt 1/%d", maxRetries),
				Message: fmt.Sprintf("Synthesizing %s...", fileName),
			})
		}

		// Initial Generation
		if !resuming {
			for attempt := 1; attempt <= maxRetries; attempt++ {
				content, err = gw.GenerateSpecFile(ctx, sess.Facts, fileName)
				if err != nil {
					if attempt == maxRetries {
						sendProgress(progress, ProgressEvent{
							File:    fileName,
							Status:  "failed",
							Details: fmt.Sprintf("failed: %v", err),
							Message: fmt.Sprintf("Failed to generate %s: %v", fileName, err),
						})
						return fmt.Errorf("failed to generate %s after %d attempts: %w", fileName, maxRetries, err)
					}
					sendProgress(progress, ProgressEvent{
						File:    fileName,
						Status:  "synthesizing",
						Details: fmt.Sprintf("error (attempt %d/%d): %v", attempt, maxRetries, err),
						Message: fmt.Sprintf("Error generating %s (attempt %d/%d): %v. Retrying...", fileName, attempt, maxRetries, err),
					})
					time.Sleep(100 * time.Millisecond)
					continue
				}
				break
			}
			// Save immediately after successful initial generation
			if err == nil {
				_ = updateInProgressState(sess, fileName, content, 1)
			}
		}

		// Self-Correction Loop for Syntax and Compliance Checks
		var complianceResults []gateway.ComplianceResult
		var checkErr error

		var applicableStds []config.Standard
		for _, std := range standards {
			for _, tf := range std.TargetFiles {
				if tf == fileName {
					applicableStds = append(applicableStds, std)
					break
				}
			}
		}

		for attempt := startAttempt; attempt < maxRetries; attempt++ {
			// Step A: Static syntax validation (YAML / JSON validation)
			checkErr = PerformStaticValidation(fileName, content)
			if checkErr != nil {
				sendProgress(progress, ProgressEvent{
					File:    fileName,
					Status:  "correcting",
					Details: fmt.Sprintf("syntax error: %v (attempt %d/%d)", checkErr, attempt+1, maxRetries),
					Message: fmt.Sprintf("⚠️ Syntax error in %s: %v. Correcting (attempt %d/%d)...", fileName, checkErr, attempt+1, maxRetries),
				})
				feedback := fmt.Sprintf("Static syntax validation failed: %v. Please rewrite the file to output syntactically valid contents.", checkErr)
				refined, refineErr := gw.RefineSpecFile(ctx, fileName, content, feedback, nil)
				if refineErr == nil {
					content = refined
					_ = updateInProgressState(sess, fileName, content, attempt+1)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Step B: Qualitative standard validation
			if len(applicableStds) > 0 {
				sendProgress(progress, ProgressEvent{
					File:    fileName,
					Status:  "auditing",
					Details: fmt.Sprintf("attempt %d/%d", attempt, maxRetries),
					Message: fmt.Sprintf("🔍 Auditing standards compliance for %s...", fileName),
				})
				evalResults, evalErr := gw.EvaluateCompliance(ctx, fileName, content, standards)
				if evalErr != nil {
					sendProgress(progress, ProgressEvent{
						File:    fileName,
						Status:  "failed",
						Details: fmt.Sprintf("compliance eval failed: %v", evalErr),
						Message: fmt.Sprintf("⚠️ Compliance evaluation failed for %s: %v", fileName, evalErr),
					})
					checkErr = evalErr
					break
				}

				complianceResults = evalResults

				// Identify failed standards
				var failedStds []config.Standard
				var feedbackLines []string
				for _, res := range complianceResults {
					var stdDef config.Standard
					for _, std := range standards {
						if std.ID == res.StandardID {
							stdDef = std
							break
						}
					}
					if !res.Compliant || res.Score < stdDef.MinScore {
						failedStds = append(failedStds, stdDef)
						feedbackLines = append(feedbackLines, fmt.Sprintf("- Standard '%s' failed (Score: %d%%, Required: %d%%): %s", stdDef.Name, res.Score, stdDef.MinScore, res.Feedback))
					}
				}

				// If failed, trigger targeted refinement
				if len(failedStds) > 0 {
					feedbackText := strings.Join(feedbackLines, "\n")
					sendProgress(progress, ProgressEvent{
						File:    fileName,
						Status:  "refining",
						Details: fmt.Sprintf("%d standards failed (attempt %d/%d)", len(failedStds), attempt+1, maxRetries),
						Message: fmt.Sprintf("🔄 Standards check failed for %s. Refining (attempt %d/%d)...", fileName, attempt+1, maxRetries),
					})
					refined, refineErr := gw.RefineSpecFile(ctx, fileName, content, feedbackText, failedStds)
					if refineErr == nil {
						content = refined
						_ = updateInProgressState(sess, fileName, content, attempt+1)
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}

			// All checks passed!
			checkErr = nil
			break
		}

		// If static syntax check failed, abort and return a hard error
		if staticErr := PerformStaticValidation(fileName, content); staticErr != nil {
			sendProgress(progress, ProgressEvent{
				File:    fileName,
				Status:  "failed",
				Details: fmt.Sprintf("syntax validation failed: %v", staticErr),
				Message: fmt.Sprintf("Syntax validation failed for %s: %v", fileName, staticErr),
			})
			return fmt.Errorf("failed to validate syntax for %s after %d attempts: %w", fileName, maxRetries, staticErr)
		}

		// Record compliance findings
		fileCompliances = append(fileCompliances, FileCompliance{
			FileName: fileName,
			Results:  complianceResults,
			Err:      checkErr,
		})

		// Write final generated content
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			sendProgress(progress, ProgressEvent{
				File:    fileName,
				Status:  "failed",
				Details: fmt.Sprintf("write failed: %v", err),
				Message: fmt.Sprintf("Failed to write %s: %v", fileName, err),
			})
			return fmt.Errorf("failed to write %s output file: %w", fileName, err)
		}

		// Update session progress
		newGenState := state.GeneratedFileState{
			FileName: fileName,
			Results:  complianceResults,
			HasError: checkErr != nil,
		}
		if checkErr != nil {
			newGenState.ErrorStr = checkErr.Error()
		}

		found := false
		for idx, gf := range sess.GeneratedFiles {
			if gf.FileName == fileName {
				sess.GeneratedFiles[idx] = newGenState
				found = true
				break
			}
		}
		if !found {
			sess.GeneratedFiles = append(sess.GeneratedFiles, newGenState)
		}

		if err := sess.Save(); err != nil {
			return fmt.Errorf("failed to save session state after generating %s: %w", fileName, err)
		}

		sendProgress(progress, ProgressEvent{
			File:    fileName,
			Status:  "done",
			Details: "completed successfully",
			Message: fmt.Sprintf("Finished generating %s", fileName),
		})
	}

	// Compile and write compliance report markdown
	sendProgress(progress, ProgressEvent{
		Status:  "compiling_report",
		Message: "Compiling compliance report (00_compliance_report.md)...",
	})
	reportContent := GenerateComplianceReport(sess.ProjectName, fileCompliances, standards)
	reportPath := filepath.Join(outputDir, "00_compliance_report.md")
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		return fmt.Errorf("failed to write 00_compliance_report.md: %w", err)
	}

	// Prepare metadata and compliance summary mapping
	complianceSummary := make(map[string]int)
	for _, fc := range fileCompliances {
		for _, res := range fc.Results {
			complianceSummary[res.StandardID] = res.Score
		}
	}

	// Generate .synthspec-meta.json
	sendProgress(progress, ProgressEvent{
		Status:  "compiling_metadata",
		Message: "Compiling solution metadata (.synthspec-meta.json)...",
	})

	meta := TelemetryMetadata{
		ProjectName:         sess.ProjectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       "1.0.0",
		ProviderUsed:        sess.Provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(sess.History) / 2,
			TokensConsumed: sess.TotalTokensUsed,
		},
		ComplianceSummary: complianceSummary,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize telemetry metadata: %w", err)
	}

	metaPath := filepath.Join(outputDir, ".synthspec-meta.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write .synthspec-meta.json: %w", err)
	}

	sendProgress(progress, ProgressEvent{
		Status:  "completed",
		Details: outputDir,
		Message: fmt.Sprintf("All files generated in: %s", outputDir),
	})
	return nil
}

func updateInProgressState(sess *state.Session, fileName, content string, attempt int) error {
	newGenState := state.GeneratedFileState{
		FileName:       fileName,
		InProgressText: content,
		CurrentAttempt: attempt,
		HasError:       true,
	}
	found := false
	for idx, gf := range sess.GeneratedFiles {
		if gf.FileName == fileName {
			newGenState.Results = gf.Results
			newGenState.ErrorStr = gf.ErrorStr
			sess.GeneratedFiles[idx] = newGenState
			found = true
			break
		}
	}
	if !found {
		sess.GeneratedFiles = append(sess.GeneratedFiles, newGenState)
	}
	return sess.Save()
}

