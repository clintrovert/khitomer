package types

import (
	"time"
)

// Task represents a Jira task with repository information
type Task struct {
	JiraTicketID    string
	Title           string
	Description     string
	Status          string
	Assignee        string
	RepositoryOwner string
	RepositoryName  string
	RepositoryURL   string
	BaseBranch      string
	CreatedAt       time.Time
}

// ProcessedTask tracks tasks that have been processed
type ProcessedTask struct {
	JiraTicketID string
	WorkflowID   string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
