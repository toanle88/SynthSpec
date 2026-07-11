package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/gateway"
)

func findSourceTemplate(templates []config.Template) (int, error) {
	for idx, template := range templates {
		if template.IsSource {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("no source template found (no template with is_source: true)")
}

func (fg *fileGenerator) generateSourceDocument(sourceTemplate config.Template, standards []config.Standard) (FileCompliance, string, string, error) {
	sendProgress(fg.progress, ProgressEvent{
		Status:  "synthesizing",
		Phase:   "source",
		File:    sourceTemplate.FileName,
		Message: fmt.Sprintf("Generating source document %s...", sourceTemplate.FileName),
	})
	sourceCompliance, err := fg.processFile(sourceTemplate.FileName, sourceTemplate.Prompt, standards, "")
	if err != nil {
		return FileCompliance{}, "", "", err
	}

	sourceDocPath := filepath.Join(fg.outputDir, sourceTemplate.FileName)
	sourceDocBytes, err := os.ReadFile(sourceDocPath)
	if err != nil {
		return FileCompliance{}, "", "", fmt.Errorf("failed to read source model document %s: %w", sourceTemplate.FileName, err)
	}

	sourceDoc := strings.TrimSpace(string(sourceDocBytes))
	if sourceDoc == "" {
		return FileCompliance{}, "", "", fmt.Errorf("source model document %s is empty", sourceTemplate.FileName)
	}

	if fg.approvalChan != nil {
		sendProgress(fg.progress, ProgressEvent{
			File:    sourceTemplate.FileName,
			Status:  "waiting_approval",
			Phase:   "source",
			Message: "Awaiting domain model approval and sign-off...",
		})
		select {
		case <-fg.approvalChan:
			// Re-read file to capture any manual edits made by the user during the pause
			sourceDocBytes, err = os.ReadFile(sourceDocPath)
			if err != nil {
				return FileCompliance{}, "", "", fmt.Errorf("failed to re-read source model document %s: %w", sourceTemplate.FileName, err)
			}
			sourceDoc = strings.TrimSpace(string(sourceDocBytes))
			sendProgress(fg.progress, ProgressEvent{
				File:    sourceTemplate.FileName,
				Status:  "done",
				Details: "completed successfully",
				Message: fmt.Sprintf("Source document %s approved and locked", sourceTemplate.FileName),
			})
		case <-fg.ctx.Done():
			return FileCompliance{}, "", "", fg.ctx.Err()
		}
	}

	sendProgress(fg.progress, ProgressEvent{
		Status:  "extracting",
		Phase:   "source",
		File:    sourceTemplate.FileName,
		Message: "Extracting dense structural entities for token optimization...",
	})
	denseEntities, err := fg.gw.ExtractStructuralEntities(fg.ctx, sourceDoc)
	if err != nil {
		return FileCompliance{}, "", "", fmt.Errorf("failed to extract structural entities: %w", err)
	}

	// Persist extracted entities to disk for auditing and verification
	entitiesPath := filepath.Join(fg.outputDir, ".synthspec-entities.json")
	if writeErr := os.WriteFile(entitiesPath, []byte(denseEntities), 0644); writeErr != nil {
		sendProgress(fg.progress, ProgressEvent{
			Status:  "warning",
			Message: fmt.Sprintf("Warning: failed to save extracted entities file: %v", writeErr),
		})
	}

	return sourceCompliance, sourceDoc, denseEntities, nil
}


type generationResult struct {
	index      int
	compliance FileCompliance
	err        error
}

func (fg *fileGenerator) generateDownstreamParallel(templates []config.Template, sourceDoc string, standards []config.Standard, fileCompliances []FileCompliance) error {
	results := make(chan generationResult, len(templates))
	var wg sync.WaitGroup

	for idx, template := range templates {
		if template.FileName == fg.sourceFileName {
			continue
		}
		wg.Add(1)
		go func(index int, t config.Template) {
			defer wg.Done()
			compliance, err := fg.processFile(t.FileName, t.Prompt, standards, sourceDoc)
			results <- generationResult{index: index, compliance: compliance, err: err}
		}(idx, template)
	}

	wg.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			return result.err
		}
		fileCompliances[result.index] = result.compliance
	}
	return nil
}

func (fg *fileGenerator) runConsistencyVerification(templates []config.Template, standards []config.Standard, fileCompliances []FileCompliance) error {
	select {
	case <-fg.forceFinishChan:
		sendProgress(fg.progress, ProgressEvent{
			Status:  "warning",
			Message: "Manual finish requested. Bypassing cross-document consistency checks.",
		})
		return fg.finishGeneration(fileCompliances, standards, &gateway.ConsistencyReport{Consistent: true})
	default:
	}

	sendProgress(fg.progress, ProgressEvent{
		Status:  "auditing",
		Message: "Verifying cross-document logical consistency...",
	})

	filesContent := fg.readFilesContent(templates)

	// Run local ConsistencyAuditor (Docs vs Docs)
	localAuditor := NewConsistencyAuditor()
	localReport, err := localAuditor.Audit(filesContent)
	if err == nil && !localReport.Consistent {
		var missing []string
		for _, feedback := range localReport.Feedback {
			if strings.Contains(feedback, "missing from 01_domain_model_use_cases.md: ") {
				parts := strings.Split(feedback, "missing from 01_domain_model_use_cases.md: ")
				if len(parts) > 1 {
					ents := strings.Split(parts[1], ", ")
					missing = append(missing, ents...)
				}
			}
		}

		if len(missing) > 0 {
			sendProgress(fg.progress, ProgressEvent{
				Status:  "correcting",
				Message: fmt.Sprintf("Retroactively updating Domain Model with missing entities: %s", strings.Join(missing, ", ")),
			})
			oldDomainModel := filesContent[fg.sourceFileName]
			updatedDomain, updateErr := ProposeUpstreamUpdate(fg.ctx, fg.gw, oldDomainModel, missing)
			if updateErr == nil {
				fg.proposedMu.Lock()
				fg.proposedContents[fg.sourceFileName] = updatedDomain
				fg.proposedMu.Unlock()
				filesContent[fg.sourceFileName] = updatedDomain
			}
		}
	}

	report, err := fg.runConsistencyRefinementLoop(filesContent, fileCompliances, standards)
	if err != nil {
		return err
	}

	return fg.finishGeneration(fileCompliances, standards, report)
}

// readFilesContent reads all currently generated template files from proposedContents or disk.
func (fg *fileGenerator) readFilesContent(templates []config.Template) map[string]string {
	filesContent := make(map[string]string)
	fg.proposedMu.Lock()
	defer fg.proposedMu.Unlock()
	for _, template := range templates {
		if content, ok := fg.proposedContents[template.FileName]; ok {
			filesContent[template.FileName] = content
			continue
		}
		filePath := filepath.Join(fg.outputDir, template.FileName)
		contentBytes, readErr := os.ReadFile(filePath)
		if readErr == nil {
			filesContent[template.FileName] = string(contentBytes)
		}
	}
	return filesContent
}

// runConsistencyRefinementLoop loops up to 3 times to evaluate consistency and auto-refine any inconsistent documents.
func (fg *fileGenerator) runConsistencyRefinementLoop(filesContent map[string]string, fileCompliances []FileCompliance, standards []config.Standard) (*gateway.ConsistencyReport, error) {
	var consistencyReport *gateway.ConsistencyReport
	for cAttempt := 1; cAttempt <= 3; cAttempt++ {
		select {
		case <-fg.forceFinishChan:
			sendProgress(fg.progress, ProgressEvent{
				Status:  "warning",
				Message: "Manual finish requested. Skipping remaining consistency refinement.",
			})
			return &gateway.ConsistencyReport{Consistent: true}, nil
		default:
		}

		report, err := fg.gw.VerifyConsistency(fg.ctx, filesContent)
		if err != nil {
			return nil, fmt.Errorf("cross-document consistency verification failed: %w", err)
		}
		consistencyReport = report

		if report.Consistent {
			break
		}

		if cAttempt == 3 {
			break
		}

		sendProgress(fg.progress, ProgressEvent{
			Status:  "correcting",
			Details: fmt.Sprintf("consistency loop %d/3", cAttempt),
			Message: fmt.Sprintf("Logical inconsistencies detected. Refining files (attempt %d/3)...", cAttempt),
		})

		for fileName, feedback := range report.Feedback {
			if err := fg.refineSingleInconsistentFile(fileName, feedback, filesContent, fileCompliances, standards); err != nil {
				return nil, err
			}
		}
	}

	if consistencyReport != nil && !consistencyReport.Consistent {
		sendProgress(fg.progress, ProgressEvent{
			Status:  "warning",
			Message: "Warning: failed to achieve cross-document consistency after 3 refinement attempts. Proceeding to save files.",
		})
	}
	return consistencyReport, nil
}

func (fg *fileGenerator) refineSingleInconsistentFile(fileName string, feedback string, filesContent map[string]string, fileCompliances []FileCompliance, standards []config.Standard) error {
	content, ok := filesContent[fileName]
	if !ok {
		return nil
	}

	var referenceDoc string
	if fileName != fg.sourceFileName {
		// Read raw source document directly for full context reference
		sourcePath := filepath.Join(fg.outputDir, fg.sourceFileName)
		if bytes, err := os.ReadFile(sourcePath); err == nil {
			referenceDoc = string(bytes)
		}
	}

	refined, err := fg.gw.RefineSpecFile(fg.ctx, fileName, content, feedback, nil, referenceDoc)
	if err != nil {
		return fmt.Errorf("failed to refine inconsistent file %s: %w", fileName, err)
	}

	fg.proposedMu.Lock()
	fg.proposedContents[fileName] = refined
	fg.proposedMu.Unlock()

	filesContent[fileName] = refined

	// Update compliance results
	evalResults, evalErr := fg.gw.EvaluateCompliance(fg.ctx, fileName, refined, standards)
	if evalErr == nil {
		fg.updateComplianceAndSession(fileName, refined, evalResults, fileCompliances)
	}
	return nil
}

func (fg *fileGenerator) finishGeneration(fileCompliances []FileCompliance, standards []config.Standard, consistencyReport *gateway.ConsistencyReport) error {
	sendProgress(fg.progress, ProgressEvent{
		Status:  "compiling_report",
		Message: "Compiling compliance report (00_compliance_report.md)...",
	})
	reportContent := GenerateComplianceReport(fg.persistence.GetProjectName(), fileCompliances, standards, consistencyReport)
	fg.proposedMu.Lock()
	fg.proposedContents["00_compliance_report.md"] = reportContent
	fg.proposedMu.Unlock()

	if err := fg.runPromptOptimization(fg.templates); err != nil {
		sendProgress(fg.progress, ProgressEvent{
			Status:  "warning",
			Message: fmt.Sprintf("Warning: prompt optimization failed: %v", err),
		})
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

	projectName := fg.persistence.GetProjectName()
	provider := fg.persistence.GetProvider()
	history := fg.persistence.GetHistory()
	totalTokens := fg.persistence.GetTotalTokens()
	currentElapsed := int64(time.Since(fg.startTime).Seconds())
	accumulatedSecs := fg.persistence.GetTotalDuration() + currentElapsed
	formattedDuration := (time.Duration(accumulatedSecs) * time.Second).String()

	meta := GenerationMetadata{
		ProjectName:         projectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       EngineVersion,
		ProviderUsed:        provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(history) / 2,
			TokensConsumed: totalTokens,
			TotalDuration:  formattedDuration,
		},
		ComplianceSummary: complianceSummary,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize telemetry metadata: %w", err)
	}

	fg.proposedMu.Lock()
	fg.proposedContents[".synthspec-meta.json"] = string(metaData)
	fg.proposedMu.Unlock()

	// Compute Diffs and wait for approval
	var diffs []domain.FileDiff
	fg.proposedMu.Lock()
	for fname, newContent := range fg.proposedContents {
		if fname == ".synthspec-meta.json" || fname == "99_optimized_prompt.md" || fname == "00_compliance_report.md" {
			continue
		}
		oldContent := ""
		filePath := filepath.Join(fg.outputDir, fname)
		if bytes, err := os.ReadFile(filePath); err == nil {
			oldContent = string(bytes)
		}
		if oldContent != newContent {
			diffs = append(diffs, ComputeDiff(fname, oldContent, newContent))
		}
	}
	fg.proposedMu.Unlock()

	if len(diffs) > 0 && fg.diffApprovalChan != nil {
		diffsJSON, _ := json.Marshal(diffs)
		sendProgress(fg.progress, ProgressEvent{
			Status:  "waiting_diff_approval",
			Details: string(diffsJSON),
			Message: "Awaiting approval for proposed file modifications...",
		})
		select {
		case <-fg.diffApprovalChan:
			// Approved, proceed
		case <-fg.ctx.Done():
			return fg.ctx.Err()
		}
	}

	// Write proposed contents to disk
	fg.proposedMu.Lock()
	for fname, newContent := range fg.proposedContents {
		filePath := filepath.Join(fg.outputDir, fname)
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			fg.proposedMu.Unlock()
			return fmt.Errorf("failed to write %s output file: %w", fname, err)
		}
	}
	fg.proposedMu.Unlock()

	sendProgress(fg.progress, ProgressEvent{
		Status:  "completed",
		Details: fg.outputDir,
		Message: fmt.Sprintf("All files generated in: %s", fg.outputDir),
	})
	return nil
}

func (fg *fileGenerator) runPromptOptimization(templates []config.Template) error {
	sendProgress(fg.progress, ProgressEvent{
		Status:  "optimizing_prompt",
		Message: "Condensing files into optimized prompt (99_optimized_prompt.md)...",
	})

	filesContent := fg.readFilesContent(templates)
	if len(filesContent) == 0 {
		return fmt.Errorf("no generated files found to optimize")
	}

	// Apply a strict 60-second timeout for prompt optimization call to prevent hanging
	optCtx, cancel := context.WithTimeout(fg.ctx, 60*time.Second)
	defer cancel()

	optimized, err := fg.gw.OptimizePrompt(optCtx, filesContent)
	if err != nil {
		return fmt.Errorf("failed to optimize prompt: %w", err)
	}

	fg.proposedMu.Lock()
	fg.proposedContents["99_optimized_prompt.md"] = optimized
	fg.proposedMu.Unlock()

	return nil
}

// writeProposedToDisk writes any currently generated in-memory file drafts to disk.
// This is used to persist progress even if the pipeline exits prematurely with an error.
func (fg *fileGenerator) writeProposedToDisk() error {
	fg.proposedMu.Lock()
	defer fg.proposedMu.Unlock()
	for fname, newContent := range fg.proposedContents {
		if fname == ".synthspec-meta.json" || fname == "99_optimized_prompt.md" || fname == "00_compliance_report.md" {
			continue
		}
		filePath := filepath.Join(fg.outputDir, fname)
		_ = os.WriteFile(filePath, []byte(newContent), 0644)
	}
	return nil
}
