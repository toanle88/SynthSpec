package security

import (
	"errors"
	"fmt"
	"regexp"
)

// SecretRule defines a pattern and a description for detecting secrets.
type SecretRule struct {
	Name    string
	Pattern *regexp.Regexp
}

var defaultRules = []SecretRule{
	{
		Name:    "AWS Access Key ID",
		Pattern: regexp.MustCompile(`(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`),
	},
	{
		Name:    "OpenAI API Key",
		Pattern: regexp.MustCompile(`sk-(proj-)?[a-zA-Z0-9]{48,80}`),
	},
	{
		Name:    "Anthropic API Key",
		Pattern: regexp.MustCompile(`sk-ant-sid01-[a-zA-Z0-9_-]{93}`),
	},
	{
		Name:    "Google API Key (Gemini)",
		Pattern: regexp.MustCompile(`AIzaSy[a-zA-Z0-9_-]{33}`),
	},
	{
		Name:    "Private Key",
		Pattern: regexp.MustCompile(`(?i)-----BEGIN[ A-Z0-9_-]*PRIVATE KEY-----`),
	},
	{
		Name:    "Generic Password/Secret Assignment",
		Pattern: regexp.MustCompile(`(?i)(api_key|apikey|secret|password|passwd|token)\s*[:=]\s*["']?[a-zA-Z0-9\-_\.\~\+\/]{16,}["']?`),
	},
	{
		Name:    "Slack Webhook URL",
		Pattern: regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`),
	},
}

// ScanForSecrets checks the input string against pre-defined secret patterns.
// If any secret is detected, it returns an error specifying which secret was found.
func ScanForSecrets(input string) error {
	for _, rule := range defaultRules {
		if rule.Pattern.MatchString(input) {
			return fmt.Errorf("pre-flight safety check failed: detected potential secret (%s)", rule.Name)
		}
	}
	return nil
}

// ErrSecretDetected represents a generic secret detection error.
var ErrSecretDetected = errors.New("potential secret detected in input")
