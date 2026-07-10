package generator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toanle/synthspec/domain"
	"github.com/toanle/synthspec/gateway"
)

func (fg *fileGenerator) getCachedFileState(fileName string) (domain.GeneratedFileState, bool) {
	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	state, found := fg.persistence.LoadGeneratedFile(fileName)
	if !found {
		return domain.GeneratedFileState{}, false
	}
	return state, true
}

func (fg *fileGenerator) updateComplianceAndSession(fileName string, refined string, evalResults []gateway.ComplianceResult, fileCompliances []FileCompliance) {
	for idx, fc := range fileCompliances {
		if fc.FileName == fileName {
			fileCompliances[idx].Results = evalResults
			// Update persistence
			state, found := fg.persistence.LoadGeneratedFile(fileName)
			if found {
				state.Results = evalResults
				state.InProgressText = refined
				_ = fg.persistence.SaveGeneratedFile(state)
			}
			break
		}
	}
}

func (fg *fileGenerator) updateSessionProgress(fileName string, promptTemplate string, complianceResults []gateway.ComplianceResult, checkErr error) error {
	currentPromptHash := computeSha256(promptTemplate)
	currentFactsHash := fg.computeFactsHash(fileName)

	newGenState := domain.GeneratedFileState{
		FileName:   fileName,
		Results:    complianceResults,
		HasError:   checkErr != nil,
		PromptHash: currentPromptHash,
		FactsHash:  currentFactsHash,
	}
	if checkErr != nil {
		newGenState.ErrMsg = checkErr.Error()
	}

	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	// Load existing state if any
	existingState, found := fg.persistence.LoadGeneratedFile(fileName)
	if found {
		newGenState.Results = existingState.Results
		newGenState.ErrMsg = existingState.ErrMsg
	}

	if err := fg.persistence.SaveGeneratedFile(newGenState); err != nil {
		return fmt.Errorf("failed to save session state after generating %s: %w", fileName, err)
	}
	return nil
}

func (fg *fileGenerator) updateInProgressState(fileName, content string, attempt int, promptTemplate string) error {
	currentPromptHash := computeSha256(promptTemplate)
	currentFactsHash := fg.computeFactsHash(fileName)

	newGenState := domain.GeneratedFileState{
		FileName:       fileName,
		InProgressText: content,
		CurrentAttempt: attempt,
		HasError:       true,
		PromptHash:     currentPromptHash,
		FactsHash:      currentFactsHash,
	}
	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	// Load existing state if any
	existingState, found := fg.persistence.LoadGeneratedFile(fileName)
	if found {
		newGenState.Results = existingState.Results
		newGenState.ErrMsg = existingState.ErrMsg
	}

	if err := fg.persistence.SaveGeneratedFile(newGenState); err != nil {
		return fmt.Errorf("failed to save in-progress state for %s: %w", fileName, err)
	}
	return nil
}

func (fg *fileGenerator) computeFactsHash(fileName string) string {
	factsBytes, _ := json.Marshal(fg.persistence.GetFacts())
	hash := computeSha256(string(factsBytes))

	if fileName != fg.sourceFileName {
		sourcePath := filepath.Join(fg.outputDir, fg.sourceFileName)
		if bytes, err := os.ReadFile(sourcePath); err == nil {
			hash = computeSha256(hash + computeSha256(string(bytes)))
		}
	}
	return hash
}

func computeSha256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
