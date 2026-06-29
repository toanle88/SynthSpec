package generator

import (
	"strings"
	"testing"
)

func TestValidateBacklog(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errSub  string
	}{
		{
			name:    "Valid backlog",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{"epics": `,
			wantErr: true,
			errSub:  "invalid JSON syntax",
		},
		{
			name:    "Empty epics",
			input:   `{"epics": []}`,
			wantErr: true,
			errSub:  "backlog must contain at least one epic",
		},
		{
			name:    "Epic missing ID",
			input:   `{"epics": [{"title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic 0 is missing ID",
		},
		{
			name:    "Epic missing Title",
			input:   `{"epics": [{"id": "EP-1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic EP-1 is missing Title",
		},
		{
			name:    "Epic missing Description",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "epic EP-1 is missing Description",
		},
		{
			name:    "Epic missing tasks",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": []}]}`,
			wantErr: true,
			errSub:  "epic EP-1 must contain at least one task",
		},
		{
			name:    "Task missing ID",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"summary": "S1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task 0 in epic EP-1 is missing ID",
		},
		{
			name:    "Task missing Summary",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "details": "Det1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 is missing Summary",
		},
		{
			name:    "Task missing Details",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "acceptance_criteria": ["AC1"]}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 is missing Details",
		},
		{
			name:    "Task missing Acceptance Criteria",
			input:   `{"epics": [{"id": "EP-1", "title": "T1", "description": "D1", "tasks": [{"id": "TSK-1", "summary": "S1", "details": "Det1", "acceptance_criteria": []}]}]}`,
			wantErr: true,
			errSub:  "task TSK-1 in epic EP-1 must contain at least one acceptance criterion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBacklog(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("expected error containing %q, got %q", tt.errSub, err.Error())
				}
			}
		})
	}
}
