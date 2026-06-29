package generator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

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

func (fg *fileGenerator) updateComplianceAndSession(fileName string, refined string, evalResults []gateway.ComplianceResult, fileCompliances []FileCompliance) {
	for idx, fc := range fileCompliances {
		if fc.FileName == fileName {
			fileCompliances[idx].Results = evalResults
			fg.sessionMu.Lock()
			for sIdx, gf := range fg.sess.GeneratedFiles {
				if gf.FileName == fileName {
					fg.sess.GeneratedFiles[sIdx].Results = evalResults
					fg.sess.GeneratedFiles[sIdx].InProgressText = refined
					break
				}
			}
			fg.sessionMu.Unlock()
			break
		}
	}
}

func (fg *fileGenerator) updateSessionProgress(fileName string, promptTemplate string, complianceResults []gateway.ComplianceResult, checkErr error) error {
	currentPromptHash := computeSha256(promptTemplate)
	currentFactsHash := fg.computeFactsHash(fileName)

	newGenState := state.GeneratedFileState{
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

func (fg *fileGenerator) updateInProgressState(fileName, content string, attempt int, promptTemplate string) error {
	currentPromptHash := computeSha256(promptTemplate)
	currentFactsHash := fg.computeFactsHash(fileName)

	newGenState := state.GeneratedFileState{
		FileName:       fileName,
		InProgressText: content,
		CurrentAttempt: attempt,
		HasError:       true,
		PromptHash:     currentPromptHash,
		FactsHash:      currentFactsHash,
	}
	fg.sessionMu.Lock()
	defer fg.sessionMu.Unlock()

	found := false
	for idx, gf := range fg.sess.GeneratedFiles {
		if gf.FileName == fileName {
			newGenState.Results = gf.Results
			newGenState.ErrMsg = gf.ErrMsg
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

func (fg *fileGenerator) computeFactsHash(fileName string) string {
	factsBytes, _ := json.Marshal(fg.sess.Facts)
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
