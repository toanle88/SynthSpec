package gateway

import (
	"encoding/json"
	"testing"
)

func TestSanitizeNextQuestion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple single question",
			input:    "What is your database engine?",
			expected: "What is your database engine?",
		},
		{
			name:     "multiple questions",
			input:    "What is your database engine? How do you scale it?",
			expected: "What is your database engine?",
		},
		{
			name:     "bullet point list",
			input:    "- What is your database engine?",
			expected: "What is your database engine?",
		},
		{
			name:     "bullet point list multiple questions",
			input:    "- What is your database engine?\n- How do you handle auth?",
			expected: "What is your database engine?",
		},
		{
			name:     "numbered list",
			input:    "1. What is your database engine?\n2. What compliance rules apply?",
			expected: "What is your database engine?",
		},
		{
			name:     "paragraph with question inside",
			input:    "To clarify the data storage, what engine do you use?",
			expected: "To clarify the data storage, what engine do you use?",
		},
		{
			name:     "non-question text",
			input:    "Please define your primary backend language",
			expected: "Please define your primary backend language",
		},
		{
			name:     "non-question text with newlines",
			input:    "\n\n  Please define your primary backend language  \nSecond line here.",
			expected: "Please define your primary backend language",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := SanitizeNextQuestion(tt.input)
			if actual != tt.expected {
				t.Errorf("SanitizeNextQuestion(%q) = %q; expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestComplianceResult_UnmarshalJSON(t *testing.T) {
	// Test unmarshalling a string (raw standard ID)
	var rawStringRes ComplianceResult
	rawStringJSON := `"clean_architecture"`
	if err := json.Unmarshal([]byte(rawStringJSON), &rawStringRes); err != nil {
		t.Fatalf("failed to unmarshal raw string ID: %v", err)
	}

	if rawStringRes.StandardID != "clean_architecture" {
		t.Errorf("expected StandardID to be 'clean_architecture', got: %s", rawStringRes.StandardID)
	}
	if rawStringRes.Score != 0 || rawStringRes.Compliant != false {
		t.Errorf("expected default values (score=0, compliant=false) for raw string ID unmarshal")
	}

	// Test unmarshalling structured object
	var structuredRes ComplianceResult
	structuredJSON := `{"standard_id": "input_validation", "score": 90, "compliant": true, "feedback": "Great validation."}`
	if err := json.Unmarshal([]byte(structuredJSON), &structuredRes); err != nil {
		t.Fatalf("failed to unmarshal structured object: %v", err)
	}

	if structuredRes.StandardID != "input_validation" {
		t.Errorf("expected StandardID 'input_validation', got: %s", structuredRes.StandardID)
	}
	if structuredRes.Score != 90 {
		t.Errorf("expected Score 90, got: %d", structuredRes.Score)
	}
	if !structuredRes.Compliant {
		t.Errorf("expected Compliant true")
	}
	if structuredRes.Feedback != "Great validation." {
		t.Errorf("expected Feedback 'Great validation.', got: %s", structuredRes.Feedback)
	}
}

