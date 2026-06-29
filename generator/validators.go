package generator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
)

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

	applicableStds := config.FilterApplicableStandards(standards, filepath.Base(filePath))
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
