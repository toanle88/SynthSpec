package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

func (fg *fileGenerator) processFile(fileName string, promptTemplate string, standards []config.Standard, referenceDoc string) (FileCompliance, error) {
	cachedState, cached := fg.getCachedFileState(fileName)
	filePath := filepath.Join(fg.outputDir, fileName)
	_, statErr := os.Stat(filePath)

	currentPromptHash := computeSha256(promptTemplate)
	currentFactsHash := fg.computeFactsHash(fileName)

	if cached && statErr == nil && !cachedState.HasError && cachedState.PromptHash == currentPromptHash && cachedState.FactsHash == currentFactsHash {
		sendProgress(fg.progress, ProgressEvent{
			File:    fileName,
			Status:  "skipped",
			Details: "already generated",
			Message: fmt.Sprintf("Skipping %s (already generated & unchanged)", fileName),
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

	content, complianceResults, checkErr, err := fg.runSelfCorrection(fileName, content, startAttempt, standards, referenceDoc, promptTemplate)
	if err != nil {
		return FileCompliance{}, err
	}

	fg.proposedMu.Lock()
	fg.proposedContents[fileName] = content
	fg.proposedMu.Unlock()

	if err := fg.updateSessionProgress(fileName, promptTemplate, complianceResults, checkErr); err != nil {
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
		fullPrompt, buildErr := buildGenerationPrompt(promptTemplate, fg.persistence.GetFacts(), referenceDoc)
		if buildErr != nil {
			return "", 0, buildErr
		}
		content, err = fg.gw.GenerateSpecFile(fg.ctx, fg.persistence.GetFacts(), fileName, fullPrompt)
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

	_ = fg.updateInProgressState(fileName, content, 1, promptTemplate)
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

func (fg *fileGenerator) runSelfCorrection(fileName string, content string, startAttempt int, standards []config.Standard, referenceDoc string, promptTemplate string) (string, []gateway.ComplianceResult, error, error) {
	var complianceResults []gateway.ComplianceResult
	var checkErr error

	applicableStds := config.FilterApplicableStandards(standards, fileName)

	for attempt := startAttempt; attempt < maxRetries; attempt++ {
		checkErr = PerformStaticValidation(fileName, content, fg.templates)
		if checkErr != nil {
			content, checkErr = fg.handleSyntaxError(fileName, content, attempt, checkErr, referenceDoc, promptTemplate)
			continue
		}

		if len(applicableStds) > 0 {
			var passed bool
			var err error
			content, complianceResults, checkErr, passed, err = fg.handleComplianceEvaluation(fileName, content, attempt, standards, referenceDoc, promptTemplate)
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

	if staticErr := PerformStaticValidation(fileName, content, fg.templates); staticErr != nil {
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

func (fg *fileGenerator) handleSyntaxError(fileName string, content string, attempt int, checkErr error, referenceDoc string, promptTemplate string) (string, error) {
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
		_ = fg.updateInProgressState(fileName, content, attempt+1, promptTemplate)
	}
	time.Sleep(100 * time.Millisecond)
	return content, checkErr
}

func (fg *fileGenerator) handleComplianceEvaluation(fileName string, content string, attempt int, standards []config.Standard, referenceDoc string, promptTemplate string) (string, []gateway.ComplianceResult, error, bool, error) {
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
			_ = fg.updateInProgressState(fileName, content, attempt+1, promptTemplate)
		}
		time.Sleep(100 * time.Millisecond)
		return content, evalResults, nil, false, nil
	}

	return content, evalResults, nil, true, nil
}
