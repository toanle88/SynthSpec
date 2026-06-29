package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/shared"
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
			actual := shared.SanitizeNextQuestion(tt.input)
			if actual != tt.expected {
				t.Errorf("shared.SanitizeNextQuestion(%q) = %q; expected %q", tt.input, actual, tt.expected)
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

func TestFilterApplicableStandards(t *testing.T) {
	standards := []config.Standard{
		{ID: "s1", TargetFiles: []string{"01_domain_model_use_cases.md"}},
		{ID: "s2", TargetFiles: []string{"02_prd_functional.md"}},
		{ID: "s3", TargetFiles: []string{"01_domain_model_use_cases.md", "03_system_architecture.md"}},
	}

	tests := []struct {
		name     string
		fileName string
		wantIDs  []string
	}{
		{"single match", "01_domain_model_use_cases.md", []string{"s1", "s3"}},
		{"only one match", "02_prd_functional.md", []string{"s2"}},
		{"no match", "04_api_architecture_integration.md", nil},
		{"empty standards", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.FilterApplicableStandards(standards, tt.fileName)
			if len(result) != len(tt.wantIDs) {
				t.Errorf("expected %d results, got %d", len(tt.wantIDs), len(result))
				return
			}
			for i, r := range result {
				if r.ID != tt.wantIDs[i] {
					t.Errorf("result[%d].ID = %q, want %q", i, r.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestFilterApplicableStandards_EmptyStandards(t *testing.T) {
	result := config.FilterApplicableStandards(nil, "any.md")
	if result != nil {
		t.Errorf("expected nil result for nil standards, got %v", result)
	}
}

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with code fences",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no code fences",
			input:    "{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "  \n  ",
			expected: "",
		},
		{
			name:     "fences without lang",
			input:    "```\nplain content\n```",
			expected: "plain content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := shared.SanitizeJSON(tt.input)
			if actual != tt.expected {
				t.Errorf("shared.SanitizeJSON(%q) = %q; expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestStreamOracleResponse(t *testing.T) {
	res := &OracleResponse{
		Facts: Facts{
			Functional: "test functional",
		},
		NextQuestion: "What next?",
	}

	tokenChan := make(chan string, 100)
	shared.StreamOracleResponse(res, tokenChan)

	// Collect all chunks
	var received strings.Builder
	for chunk := range tokenChan {
		received.WriteString(chunk)
	}

	// Verify it's valid JSON that matches the original
	var decoded OracleResponse
	if err := json.Unmarshal([]byte(received.String()), &decoded); err != nil {
		t.Fatalf("streamed output should be valid JSON: %v\nGot: %s", err, received.String())
	}
	if decoded.NextQuestion != "What next?" {
		t.Errorf("expected NextQuestion 'What next?', got %q", decoded.NextQuestion)
	}
}

func TestStreamOracleResponse_EmptyResponse(t *testing.T) {
	res := &OracleResponse{}
	tokenChan := make(chan string, 100)
	shared.StreamOracleResponse(res, tokenChan)

	var received strings.Builder
	for chunk := range tokenChan {
		received.WriteString(chunk)
	}

	if received.Len() == 0 {
		t.Error("expected some output for empty response")
	}
}
