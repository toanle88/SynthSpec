package gateway

import "testing"

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
