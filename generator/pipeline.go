package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/toanle/synthspec/config"
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

func (fg *fileGenerator) generateSourceDocument(sourceTemplate config.Template, standards []config.Standard) (FileCompliance, string, error) {
	sendProgress(fg.progress, ProgressEvent{
		Status:  "synthesizing",
		Phase:   "source",
		File:    sourceTemplate.FileName,
		Message: fmt.Sprintf("Generating source document %s...", sourceTemplate.FileName),
	})
	sourceCompliance, err := fg.processFile(sourceTemplate.FileName, sourceTemplate.Prompt, standards, "")
	if err != nil {
		return FileCompliance{}, "", err
	}

	sourceDocPath := filepath.Join(fg.outputDir, sourceTemplate.FileName)
	sourceDocBytes, err := os.ReadFile(sourceDocPath)
	if err != nil {
		return FileCompliance{}, "", fmt.Errorf("failed to read source model document %s: %w", sourceTemplate.FileName, err)
	}

	sourceDoc := strings.TrimSpace(string(sourceDocBytes))
	if sourceDoc == "" {
		return FileCompliance{}, "", fmt.Errorf("source model document %s is empty", sourceTemplate.FileName)
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
				return FileCompliance{}, "", fmt.Errorf("failed to re-read source model document %s: %w", sourceTemplate.FileName, err)
			}
			sourceDoc = strings.TrimSpace(string(sourceDocBytes))
			sendProgress(fg.progress, ProgressEvent{
				File:    sourceTemplate.FileName,
				Status:  "done",
				Details: "completed successfully",
				Message: fmt.Sprintf("Source document %s approved and locked", sourceTemplate.FileName),
			})
		case <-fg.ctx.Done():
			return FileCompliance{}, "", fg.ctx.Err()
		}
	}

	return sourceCompliance, sourceDoc, nil
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
	sendProgress(fg.progress, ProgressEvent{
		Status:  "auditing",
		Message: "Verifying cross-document logical consistency...",
	})

	filesContent := fg.readFilesContent(templates)

	report, err := fg.runConsistencyRefinementLoop(filesContent, fileCompliances, standards)
	if err != nil {
		return err
	}

	return fg.finishGeneration(fileCompliances, standards, report)
}

// readFilesContent reads all currently generated template files from disk and returns their text contents.
func (fg *fileGenerator) readFilesContent(templates []config.Template) map[string]string {
	filesContent := make(map[string]string)
	for _, template := range templates {
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
		return nil, fmt.Errorf("failed to achieve cross-document consistency after 3 refinement attempts")
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
		sourcePath := filepath.Join(fg.outputDir, fg.sourceFileName)
		if bytes, err := os.ReadFile(sourcePath); err == nil {
			referenceDoc = string(bytes)
		}
	}

	refined, err := fg.gw.RefineSpecFile(fg.ctx, fileName, content, feedback, nil, referenceDoc)
	if err != nil {
		return fmt.Errorf("failed to refine inconsistent file %s: %w", fileName, err)
	}

	filePath := filepath.Join(fg.outputDir, fileName)
	if err := os.WriteFile(filePath, []byte(refined), 0644); err != nil {
		return fmt.Errorf("failed to save refined content for %s: %w", fileName, err)
	}

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

	// Get session info from persistence
	projectName := fg.persistence.GetProjectName()
	provider := fg.persistence.GetProvider()
	history := fg.persistence.GetHistory()
	totalTokens := fg.persistence.GetTotalTokens()

	meta := GenerationMetadata{
		ProjectName:         projectName,
		GenerationTimestamp: time.Now().Format(time.RFC3339),
		EngineVersion:       EngineVersion,
		ProviderUsed:        provider,
		CompletionMetrics: CompletionMetrics{
			TotalTurns:     len(history) / 2,
			TokensConsumed: totalTokens,
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
