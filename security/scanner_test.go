package security

import (
	"testing"
)

func TestScanForSecrets(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "Safe input",
			input:     "We need to implement a user login module that hashes passwords securely using bcrypt.",
			expectErr: false,
		},
		{
			name:      "AWS Key ID",
			input:     "My key is AKIAIOSFODNN7EXAMPLE, don't share it.",
			expectErr: true,
		},
		{
			name:      "OpenAI API Key legacy/standard",
			input:     "sk-Uj1234567890123456789012345678901234567890123456",
			expectErr: true,
		},
		{
			name:      "OpenAI Project Key",
			input:     "The token sk-proj-12345678901234567890123456789012345678901234567890123456 is active.",
			expectErr: true,
		},
		{
			name:      "Anthropic Key",
			input:     "sk-ant-sid01-123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890-123456",
			expectErr: true,
		},
		{
			name:      "Gemini API Key",
			input:     "AIzaSyD_u8X9asD0-asD123asD123asD123asD1",
			expectErr: true,
		},
		{
			name:      "Private Key PEM block",
			input:     "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0Y...\n-----END RSA PRIVATE KEY-----",
			expectErr: true,
		},
		{
			name:      "Generic API Key Assignment",
			input:     "api_key = \"supersecret123456789\"",
			expectErr: true,
		},
		{
			name:      "Generic Token Assignment",
			input:     "token: mysecuretokenstring123",
			expectErr: true,
		},
		{
			name:      "Slack Webhook URL",
			input:     "https://hooks.slack.com/services/" + "T12345678/B12345678/abcdefghijklmnopqrstuvwx",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ScanForSecrets(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ScanForSecrets(%q) error = %v, expectErr = %v", tt.input, err, tt.expectErr)
			}
		})
	}
}
