package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

const sourceModelFileName = "01_domain_model_use_cases.md"
const maxRetries = 10

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
	Phase   string `json:"phase,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
	ValLogs string `json:"val_logs,omitempty"`
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
		content = strings.TrimSuffix(content, "```")
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
		if err := validateEpic(epic, i); err != nil {
			return err
		}
	}
	return nil
}

func validateEpic(epic Epic, index int) error {
	if epic.ID == "" {
		return fmt.Errorf("epic %d is missing ID", index)
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
		if err := validateTask(task, j, epic.ID); err != nil {
			return err
		}
	}
	return nil
}

func validateTask(task Task, index int, epicID string) error {
	if task.ID == "" {
		return fmt.Errorf("task %d in epic %s is missing ID", index, epicID)
	}
	if task.Summary == "" {
		return fmt.Errorf("task %s in epic %s is missing Summary", task.ID, epicID)
	}
	if task.Details == "" {
		return fmt.Errorf("task %s in epic %s is missing Details", task.ID, epicID)
	}
	if len(task.AcceptanceCriteria) == 0 {
		return fmt.Errorf("task %s in epic %s must contain at least one acceptance criterion", task.ID, epicID)
	}
	return nil
}

// Generate runs sequential spec generation for all files
type fileGenerator struct {
	ctx       context.Context
	gw        gateway.Gateway
	sess      *state.Session
	outputDir string
	progress  chan<- string
	sessionMu sync.Mutex
}

// Generate runs sequential spec generation for all files
func Generate(ctx context.Context, gw gateway.Gateway, sess *state.Session, outputDir string, progress chan<- string) error {
	defer close(progress)

	// Load templates
	templates, err := config.LoadTemplates()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

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

	var files []string
	for _, t := range templates {
		files = append(files, t.FileName)
	}

	sendProgress(progress, ProgressEvent{Status: "started", Phase: "source", Details: strings.Join(files, ","), Message: "Starting spec generation..."})

	fg := &fileGenerator{
		ctx:       ctx,
		gw:        gw,
		sess:      sess,
		outputDir: outputDir,
		progress:  progress,
	}

	fileCompliances := make([]FileCompliance, len(templates))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	fg.ctx = runCtx

	sourceTemplateIndex := -1
	for idx, template := range templates {
		if template.FileName == sourceModelFileName {
			sourceTemplateIndex = idx
			break
		}
	}
	if sourceTemplateIndex == -1 {
		return fmt.Errorf("source model template %q not found", sourceModelFileName)
	}

	sourceTemplate := templates[sourceTemplateIndex]
	sendProgress(progress, ProgressEvent{
		Status:  "started",
		Phase:   "source",
		File:    sourceTemplate.FileName,
		Message: fmt.Sprintf("Generating source document %s...", sourceTemplate.FileName),
	})
	sourceCompliance, err := fg.processFile(sourceTemplate.FileName, sourceTemplate.Prompt, standards, "")
	if err != nil {
		cancel()
		return err
	}
	fileCompliances[sourceTemplateIndex] = sourceCompliance

	sourceDocPath := filepath.Join(outputDir, sourceTemplate.FileName)
	sourceDocBytes, err := os.ReadFile(sourceDocPath)
	if err != nil {
		return fmt.Errorf("failed to read source model document %s: %w", sourceTemplate.FileName, err)
	}
	sourceDoc := strings.TrimSpace(string(sourceDocBytes))
	if sourceDoc == "" {
		return fmt.Errorf("source model document %s is empty", sourceTemplate.FileName)
	}

	sendProgress(progress, ProgressEvent{
		Status:  "started",
		Phase:   "parallel",
		Details: strings.Join(files, ","),
		Message: fmt.Sprintf("Source document locked. Starting parallel generation for %d downstream documents...", len(templates)-1),
	})

	type generationResult struct {
		index      int
		compliance FileCompliance
		err        error
	}

	type downstreamTemplate struct {
		index    int
		template config.Template
	}

	downstreamTemplates := make([]downstreamTemplate, 0, len(templates)-1)
	for idx, template := range templates {
		if template.FileName == sourceModelFileName {
			continue
		}
		downstreamTemplates = append(downstreamTemplates, downstreamTemplate{index: idx, template: template})
	}

	results := make(chan generationResult, len(downstreamTemplates))
	var wg sync.WaitGroup
	for _, item := range downstreamTemplates {
		wg.Add(1)
		go func(index int, template config.Template) {
			defer wg.Done()

			compliance, err := fg.processFile(template.FileName, template.Prompt, standards, sourceDoc)
			results <- generationResult{index: index, compliance: compliance, err: err}
			if err != nil {
				cancel()
			}
		}(item.index, item.template)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var firstErr error
	for result := range results {
		if result.err != nil {
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		fileCompliances[result.index] = result.compliance
	}

	if firstErr != nil {
		return firstErr
	}

	return fg.finishGeneration(fileCompliances, standards)
}

func (fg *fileGenerator) getCachedFileState(fileName string) (state.GeneratedFileState, bool) {
	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	for _, gf := range fg.sess.GeneratedFiles {
		if gf.FileName == fileName {
			return gf, true
		}
	}
	return state.GeneratedFileState{}, false
}

func (fg *fileGenerator) processFile(fileName string, promptTemplate string, standards []config.Standard, referenceDoc string) (FileCompliance, error) {
	cachedState, cached := fg.getCachedFileState(fileName)
	filePath := filepath.Join(fg.outputDir, fileName)
	_, statErr := os.Stat(filePath)

	if cached && statErr == nil && !cachedState.HasError {
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "skipped",
			Details: "already generated",
			Message: fmt.Sprintf("Skipping %s (already generated)", fileName),
		})
		return FileCompliance{
			FileName: fileName,
			Results:  cachedState.Results,
			Err:      nil,
		}, nil
	}

	content, startAttempt, err := fg.getInitialContentOrResume(fileName, promptTemplate, referenceDoc)
	if err != nil {
		return FileCompliance{}, err
	}

	content, complianceResults, checkErr, err := fg.runSelfCorrection(fileName, content, startAttempt, standards, referenceDoc)
	if err != nil {
		return FileCompliance{}, err
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "failed",
			Details: fmt.Sprintf("write failed: %v", err),
			Message: fmt.Sprintf("Failed to write %s: %v", fileName, err),
		})
		return FileCompliance{}, fmt.Errorf("failed to write %s output file: %w", fileName, err)
	}

	if err := fg.updateSessionProgress(fileName, complianceResults, checkErr); err != nil {
		return FileCompliance{}, err
	}

	sendProgress(fg.progress, ProgressEvent{
		File:    fileName,
		Status:  "done",
		Details: "completed successfully",
		Message: fmt.Sprintf("Finished generating %s", fileName),
	})

	return FileCompliance{
		FileName: fileName,
		Results:  complianceResults,
		Err:      checkErr,
	}, nil
}

func (fg *fileGenerator) getInitialContentOrResume(fileName string, promptTemplate string, referenceDoc string) (string, int, error) {
	cachedState, cached := fg.getCachedFileState(fileName)

	if cached && cachedState.InProgressText != "" {
		content := cachedState.InProgressText
		startAttempt := cachedState.CurrentAttempt
		if startAttempt < 1 {
			startAttempt = 1
		}
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "synthesizing",
			Details: fmt.Sprintf("resuming from attempt %d/%d", startAttempt, maxRetries),
			Message: fmt.Sprintf("Resuming %s synthesis from attempt %d...", fileName, startAttempt),
		})
		return content, startAttempt, nil
	}

	sendProgress(fg.progress, ProgressEvent{
		File:    fileName,
		Status:  "synthesizing",
		Details: fmt.Sprintf("attempt 1/%d", maxRetries),
		Message: fmt.Sprintf("Synthesizing %s...", fileName),
	})

	var content string
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fullPrompt, buildErr := buildGenerationPrompt(promptTemplate, fg.sess.Facts, referenceDoc)
		if buildErr != nil {
			return "", 0, buildErr
		}
		content, err = fg.gw.GenerateSpecFile(fg.ctx, fg.sess.Facts, fileName, fullPrompt)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			sendProgress(fg.progress, ProgressEvent{
				File:    fileName,
				Status:  "failed",
				Details: fmt.Sprintf("failed: %v", err),
				Message: fmt.Sprintf("Failed to generate %s: %v", fileName, err),
			})
			return "", 0, fmt.Errorf("failed to generate %s after %d attempts: %w", fileName, maxRetries, err)
		}
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "synthesizing",
			Details: fmt.Sprintf("error (attempt %d/%d): %v", attempt, maxRetries, err),
			Message: fmt.Sprintf("Error generating %s (attempt %d/%d): %v. Retrying...", fileName, attempt, maxRetries, err),
		})
		time.Sleep(100 * time.Millisecond)
	}

	_ = fg.updateInProgressState(fileName, content, 1)
	return content, 1, nil
}

func buildGenerationPrompt(promptTemplate string, facts gateway.Facts, referenceDoc string) (string, error) {
	factsJSON, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal facts for prompt: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(promptTemplate))
	builder.WriteString("\n")
	builder.WriteString(string(factsJSON))

	if trimmedReference := strings.TrimSpace(referenceDoc); trimmedReference != "" {
		builder.WriteString("\n\n")
		builder.WriteString("Reference source document:\n")
		builder.WriteString(trimmedReference)
	}

	return builder.String(), nil
}

func (fg *fileGenerator) runSelfCorrection(fileName string, content string, startAttempt int, standards []config.Standard, referenceDoc string) (string, []gateway.ComplianceResult, error, error) {
	var complianceResults []gateway.ComplianceResult
	var checkErr error

	applicableStds := getApplicableStandards(standards, fileName)

	for attempt := startAttempt; attempt < maxRetries; attempt++ {
		checkErr = PerformStaticValidation(fileName, content)
		if checkErr != nil {
			content, checkErr = fg.handleSyntaxError(fileName, content, attempt, checkErr, referenceDoc)
			continue
		}

		if len(applicableStds) > 0 {
			var passed bool
			var err error
			content, complianceResults, checkErr, passed, err = fg.handleComplianceEvaluation(fileName, content, attempt, standards, referenceDoc)
			if err != nil {
				return content, nil, nil, err
			}
			if !passed {
				continue
			}
		}

		checkErr = nil
		break
	}

	if staticErr := PerformStaticValidation(fileName, content); staticErr != nil {
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "failed",
			Details: fmt.Sprintf("syntax validation failed: %v", staticErr),
			Message: fmt.Sprintf("Syntax validation failed for %s: %v", fileName, staticErr),
		})
		return content, nil, nil, fmt.Errorf("failed to validate syntax for %s after %d attempts: %w", fileName, maxRetries, staticErr)
	}

	return content, complianceResults, checkErr, nil
}

func getApplicableStandards(standards []config.Standard, fileName string) []config.Standard {
	var applicableStds []config.Standard
	for _, std := range standards {
		for _, tf := range std.TargetFiles {
			if tf == fileName {
				applicableStds = append(applicableStds, std)
				break
			}
		}
	}
	return applicableStds
}

func (fg *fileGenerator) handleSyntaxError(fileName string, content string, attempt int, checkErr error, referenceDoc string) (string, error) {
	sendProgress(fg.progress, ProgressEvent{
		File:    fileName,
		Status:  "correcting",
		Details: fmt.Sprintf("syntax error: %v (attempt %d/%d)", checkErr, attempt+1, maxRetries),
		Message: fmt.Sprintf("⚠️ Syntax error in %s: %v. Correcting (attempt %d/%d)...", fileName, checkErr, attempt+1, maxRetries),
	})
	feedback := fmt.Sprintf("Static syntax validation failed: %v. Please rewrite the file to output syntactically valid contents.", checkErr)
	refined, refineErr := fg.gw.RefineSpecFile(fg.ctx, fileName, content, feedback, nil, referenceDoc)
	if refineErr == nil {
		content = refined
		_ = fg.updateInProgressState(fileName, content, attempt+1)
	}
	time.Sleep(100 * time.Millisecond)
	return content, checkErr
}

func runExternalValidator(ctx context.Context, cmdStr string, filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err == nil {
		filePath = absPath
	}

	if strings.Contains(cmdStr, "{path}") {
		cmdStr = strings.ReplaceAll(cmdStr, "{path}", filePath)
	} else {
		cmdStr = cmdStr + " " + filePath
	}

	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd.exe", "/c", cmdStr)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", cmdStr)
	}

	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after 10 seconds")
		}
		return output, err
	}
	return output, nil
}

func updateComplianceResultWithValidationError(res *gateway.ComplianceResult, valErr error, valOutput string) {
	if valErr == nil {
		return
	}
	res.Compliant = false
	res.Score = 0
	errorMsg := valErr.Error()
	if strings.TrimSpace(valOutput) != "" {
		errorMsg = strings.TrimSpace(valOutput)
	}
	if res.Feedback != "" {
		res.Feedback = fmt.Sprintf("%s\nExternal validator failed:\n%s", res.Feedback, errorMsg)
	} else {
		res.Feedback = fmt.Sprintf("External validator failed:\n%s", errorMsg)
	}
}

func getOrInsertResult(stdID string, resultsMap map[string]*gateway.ComplianceResult, evalResults *[]gateway.ComplianceResult) *gateway.ComplianceResult {
	res, exists := resultsMap[stdID]
	if !exists {
		newRes := gateway.ComplianceResult{
			StandardID: stdID,
			Score:      100,
			Compliant:  true,
		}
		*evalResults = append(*evalResults, newRes)
		res = &(*evalResults)[len(*evalResults)-1]
		resultsMap[stdID] = res
	}
	return res
}

func (fg *fileGenerator) runExternalValidators(evalResults []gateway.ComplianceResult, standards []config.Standard, filePath string) ([]gateway.ComplianceResult, error) {
	resultsMap := make(map[string]*gateway.ComplianceResult)
	for i := range evalResults {
		resultsMap[evalResults[i].StandardID] = &evalResults[i]
	}

	applicableStds := getApplicableStandards(standards, filepath.Base(filePath))
	for _, std := range applicableStds {
		if std.ValidatorCmd == "" {
			continue
		}

		sendProgress(fg.progress, ProgressEvent{
			File:    filepath.Base(filePath),
			Status:  "auditing",
			ValLogs: fmt.Sprintf("[%s] Running validator: %s", std.ID, std.ValidatorCmd),
		})

		valOutput, valErr := runExternalValidator(fg.ctx, std.ValidatorCmd, filePath)

		statusMsg := "SUCCESS"
		if valErr != nil {
			statusMsg = "FAILED"
		}
		logContent := fmt.Sprintf("[%s] Status: %s", std.ID, statusMsg)
		if strings.TrimSpace(valOutput) != "" {
			logContent = fmt.Sprintf("%s\n%s", logContent, strings.TrimSpace(valOutput))
		}
		sendProgress(fg.progress, ProgressEvent{
			File:    filepath.Base(filePath),
			Status:  "auditing",
			ValLogs: logContent,
		})

		res := getOrInsertResult(std.ID, resultsMap, &evalResults)
		updateComplianceResultWithValidationError(res, valErr, valOutput)
	}
	return evalResults, nil
}

func collectFailedStandards(evalResults []gateway.ComplianceResult, standards []config.Standard) ([]config.Standard, []string) {
	var failedStds []config.Standard
	var feedbackLines []string
	for _, res := range evalResults {
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
	return failedStds, feedbackLines
}

func (fg *fileGenerator) handleComplianceEvaluation(fileName string, content string, attempt int, standards []config.Standard, referenceDoc string) (string, []gateway.ComplianceResult, error, bool, error) {
	sendProgress(fg.progress, ProgressEvent{
		File:    fileName,
		Status:  "auditing",
		Details: fmt.Sprintf("attempt %d/%d", attempt, maxRetries),
		Message: fmt.Sprintf("🔍 Auditing standards compliance for %s...", fileName),
	})

	// Write the current draft content to the actual file path first so that external validators can check it
	filePath := filepath.Join(fg.outputDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return content, nil, err, false, err
	}

	evalResults, evalErr := fg.gw.EvaluateCompliance(fg.ctx, fileName, content, standards)
	if evalErr != nil {
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "failed",
			Details: fmt.Sprintf("compliance eval failed: %v", evalErr),
			Message: fmt.Sprintf("⚠️ Compliance evaluation failed for %s: %v", fileName, evalErr),
		})
		return content, nil, evalErr, false, evalErr
	}

	evalResults, err := fg.runExternalValidators(evalResults, standards, filePath)
	if err != nil {
		return content, nil, err, false, err
	}

	failedStds, feedbackLines := collectFailedStandards(evalResults, standards)

	if len(failedStds) > 0 {
		feedbackText := strings.Join(feedbackLines, "\n")
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "refining",
			Details: fmt.Sprintf("%d standards failed (attempt %d/%d)", len(failedStds), attempt+1, maxRetries),
			Message: fmt.Sprintf("🔄 Standards check failed for %s. Refining (attempt %d/%d)...", fileName, attempt+1, maxRetries),
		})
		refined, refineErr := fg.gw.RefineSpecFile(fg.ctx, fileName, content, feedbackText, failedStds, referenceDoc)
		if refineErr == nil {
			content = refined
			_ = fg.updateInProgressState(fileName, content, attempt+1)
		}
		time.Sleep(100 * time.Millisecond)
		return content, evalResults, nil, false, nil
	}

	return content, evalResults, nil, true, nil
}

func (fg *fileGenerator) updateSessionProgress(fileName string, complianceResults []gateway.ComplianceResult, checkErr error) error {
	newGenState := state.GeneratedFileState{
		FileName: fileName,
		Results:  complianceResults,
		HasError: checkErr != nil,
	}
	if checkErr != nil {
		newGenState.ErrorStr = checkErr.Error()
	}

	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	found := false
	for idx, gf := range fg.sess.GeneratedFiles {
		if gf.FileName == fileName {
			fg.sess.GeneratedFiles[idx] = newGenState
			found = true
			break
		}
	}
	if !found {
		fg.sess.GeneratedFiles = append(fg.sess.GeneratedFiles, newGenState)
	}

	if err := fg.sess.Save(); err != nil {
		return fmt.Errorf("failed to save session state after generating %s: %w", fileName, err)
	}
	return nil
}

func (fg *fileGenerator) finishGeneration(fileCompliances []FileCompliance, standards []config.Standard) error {
	sendProgress(fg.progress, ProgressEvent{
		Status:  "compiling_report",
		Message: "Compiling compliance report (00_compliance_report.md)...",
	})
	reportContent := GenerateComplianceReport(fg.sess.ProjectName, fileCompliances, standards)
	reportPath := filepath.Join(fg.outputDir, "00_compliance_report.md")
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		return fmt.Errorf("failed to write 00_compliance_report.md: %w", err)
	}

	complianceSummary := make(map[string]int)
	for _, fc := range fileCompliances {
		for _, res := range fc.Results {
			complianceSummary[res.StandardID] = res.Score
		}
	}

	sendProgress(fg.progress, ProgressEvent{
		Status:  "compiling_metadata",
		Message: "Compiling solution metadata (.synthspec-meta.json)...",
	})

	meta := TelemetryMetadata{
		ProjectName:         fg.sess.ProjectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       "1.0.0",
		ProviderUsed:        fg.sess.Provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(fg.sess.History) / 2,
			TokensConsumed: fg.sess.TotalTokensUsed,
		},
		ComplianceSummary: complianceSummary,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize telemetry metadata: %w", err)
	}

	metaPath := filepath.Join(fg.outputDir, ".synthspec-meta.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write .synthspec-meta.json: %w", err)
	}

	sendProgress(fg.progress, ProgressEvent{
		Status:  "completed",
		Details: fg.outputDir,
		Message: fmt.Sprintf("All files generated in: %s", fg.outputDir),
	})
	return nil
}

func (fg *fileGenerator) updateInProgressState(fileName, content string, attempt int) error {
	newGenState := state.GeneratedFileState{
		FileName:       fileName,
		InProgressText: content,
		CurrentAttempt: attempt,
		HasError:       true,
	}
	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	found := false
	for idx, gf := range fg.sess.GeneratedFiles {
		if gf.FileName == fileName {
			newGenState.Results = gf.Results
			newGenState.ErrorStr = gf.ErrorStr
			fg.sess.GeneratedFiles[idx] = newGenState
			found = true
			break
		}
	}
	if !found {
		fg.sess.GeneratedFiles = append(fg.sess.GeneratedFiles, newGenState)
	}
	return fg.sess.Save()
}
