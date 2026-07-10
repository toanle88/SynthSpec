package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComplianceResult_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ComplianceResult
		wantErr  bool
	}{
		{
			name:  "String fallback",
			input: `"clean_architecture"`,
			expected: ComplianceResult{
				StandardID: "clean_architecture",
				Score:      0,
				Compliant:  false,
				Feedback:   "Auditor returned only standard ID without detailed metrics.",
			},
			wantErr: false,
		},
		{
			name:  "Structured object",
			input: `{"standard_id": "solid", "score": 85, "compliant": true, "feedback": "Good"}`,
			expected: ComplianceResult{
				StandardID: "solid",
				Score:      85,
				Compliant:  true,
				Feedback:   "Good",
			},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ComplianceResult
			err := json.Unmarshal([]byte(tt.input), &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if result.StandardID != tt.expected.StandardID {
					t.Errorf("StandardID = %v, want %v", result.StandardID, tt.expected.StandardID)
				}
				if result.Score != tt.expected.Score {
					t.Errorf("Score = %v, want %v", result.Score, tt.expected.Score)
				}
				if result.Compliant != tt.expected.Compliant {
					t.Errorf("Compliant = %v, want %v", result.Compliant, tt.expected.Compliant)
				}
				if result.Feedback != tt.expected.Feedback {
					t.Errorf("Feedback = %v, want %v", result.Feedback, tt.expected.Feedback)
				}
			}
		})
	}
}

func TestStreamOracleResponse(t *testing.T) {
	res := &OracleResponse{
		NextQuestion: "What is your quest?",
	}

	tokenChan := make(chan string)
	
	StreamOracleResponse(res, tokenChan)

	var received string
	
	// Collect all tokens until channel is closed
	done := make(chan bool)
	go func() {
		for chunk := range tokenChan {
			received += chunk
		}
		done <- true
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for stream")
	case <-done:
	}
	
	if len(received) == 0 {
		t.Errorf("Expected to receive chunks, got empty string")
	}
}
