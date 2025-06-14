package domain

import "time"

type WorkflowStatus string

type Workflow struct {
	ID          string
	Name        string
	Description string
	Trigger     Trigger
	Steps       []WorkflowStep
	Variables   map[string]interface{}
	Status      WorkflowStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}