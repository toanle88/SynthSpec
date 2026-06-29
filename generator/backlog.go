package generator

import (
	"encoding/json"
	"fmt"

	"github.com/toanle/synthspec/shared"
)

// Backlog represents the top-level structure of the engineering backlog
type Backlog struct {
	Epics []Epic `json:"epics"`
}

// Epic represents a high-level feature category containing tasks
type Epic struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tasks       []Task `json:"tasks"`
}

// Task represents a development task in the backlog
type Task struct {
	ID                 string   `json:"id"`
	Summary            string   `json:"summary"`
	Details            string   `json:"details"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// validateBacklog parses and validates the engineering backlog JSON against structural requirements
func validateBacklog(content string) error {
	content = shared.SanitizeJSON(content)
	var backlog Backlog
	if err := json.Unmarshal([]byte(content), &backlog); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	if len(backlog.Epics) == 0 {
		return fmt.Errorf("backlog must contain at least one epic")
	}

	for i, epic := range backlog.Epics {
		if err := validateEpic(epic, i); err != nil {
			return err
		}
	}
	return nil
}

func validateEpic(epic Epic, index int) error {
	if epic.ID == "" {
		return fmt.Errorf("epic %d is missing ID", index)
	}
	if epic.Title == "" {
		return fmt.Errorf("epic %s is missing Title", epic.ID)
	}
	if epic.Description == "" {
		return fmt.Errorf("epic %s is missing Description", epic.ID)
	}
	if len(epic.Tasks) == 0 {
		return fmt.Errorf("epic %s must contain at least one task", epic.ID)
	}
	for j, task := range epic.Tasks {
		if err := validateTask(task, j, epic.ID); err != nil {
			return err
		}
	}
	return nil
}

func validateTask(task Task, index int, epicID string) error {
	if task.ID == "" {
		return fmt.Errorf("task %d in epic %s is missing ID", index, epicID)
	}
	if task.Summary == "" {
		return fmt.Errorf("task %s in epic %s is missing Summary", task.ID, epicID)
	}
	if task.Details == "" {
		return fmt.Errorf("task %s in epic %s is missing Details", task.ID, epicID)
	}
	if len(task.AcceptanceCriteria) == 0 {
		return fmt.Errorf("task %s in epic %s must contain at least one acceptance criterion", task.ID, epicID)
	}
	return nil
}
