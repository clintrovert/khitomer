package jira

import (
	"fmt"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/pkg/types"
)

// Client wraps Jira API client functionality
type Client struct {
	client      *jira.Client
	logger      *zap.Logger
	projectKey  string
	customField string
}

// NewClient creates a new Jira client
func NewClient(baseURL, username, apiToken, projectKey, customField string, logger *zap.Logger) (*Client, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: apiToken,
	}

	client, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create jira client: %w", err)
	}

	return &Client{
		client:      client,
		logger:      logger,
		projectKey:  projectKey,
		customField: customField,
	}, nil
}

// GetTasksByStatus retrieves tasks with a specific status
func (c *Client) GetTasksByStatus(status string) ([]*types.Task, error) {
	jql := fmt.Sprintf("project = %s AND status = \"%s\"", c.projectKey, status)
	
	issues, _, err := c.client.Issue.Search(jql, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	tasks := make([]*types.Task, 0, len(issues))
	for _, issue := range issues {
		task, err := c.issueToTask(&issue)
		if err != nil {
			c.logger.Warn("failed to convert issue to task", zap.Error(err), zap.String("issue", issue.Key))
			continue
		}
		if task != nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// GetTask retrieves a specific task by ID
func (c *Client) GetTask(ticketID string) (*types.Task, error) {
	issue, _, err := c.client.Issue.Get(ticketID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	return c.issueToTask(issue)
}

// UpdateTaskStatus updates the status of a task
func (c *Client) UpdateTaskStatus(ticketID, status string) error {
	transitions, _, err := c.client.Issue.GetTransitions(ticketID)
	if err != nil {
		return fmt.Errorf("failed to get transitions: %w", err)
	}

	var transitionID string
	for _, transition := range transitions {
		if strings.EqualFold(transition.To.Name, status) {
			transitionID = transition.ID
			break
		}
	}

	if transitionID == "" {
		return fmt.Errorf("transition to status %s not found", status)
	}

	_, err = c.client.Issue.DoTransition(ticketID, transitionID)
	if err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}

	return nil
}

// AddComment adds a comment to a task
func (c *Client) AddComment(ticketID, comment string) error {
	_, _, err := c.client.Issue.AddComment(ticketID, &jira.Comment{
		Body: comment,
	})
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// issueToTask converts a Jira issue to a Task
func (c *Client) issueToTask(issue *jira.Issue) (*types.Task, error) {
	// Extract repository information from custom field
	repoOwner, repoName, err := c.extractRepositoryInfo(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to extract repository info: %w", err)
	}

	if repoOwner == "" || repoName == "" {
		// Skip tasks without repository information
		return nil, nil
	}

	task := &types.Task{
		JiraTicketID:     issue.Key,
		Title:           issue.Fields.Summary,
		Description:     issue.Fields.Description,
		Status:          issue.Fields.Status.Name,
		RepositoryOwner: repoOwner,
		RepositoryName:  repoName,
		RepositoryURL:   fmt.Sprintf("https://github.com/%s/%s", repoOwner, repoName),
		BaseBranch:      "main", // Default, can be overridden
	}

	if issue.Fields.Assignee != nil {
		task.Assignee = issue.Fields.Assignee.DisplayName
	}

	return task, nil
}

// extractRepositoryInfo extracts repository owner and name from custom field
func (c *Client) extractRepositoryInfo(issue *jira.Issue) (string, string, error) {
	// Try to find the custom field by name
	for key, value := range issue.Fields.Unknowns {
		if strings.Contains(strings.ToLower(key), strings.ToLower(c.customField)) {
			repoStr, ok := value.(string)
			if !ok {
				continue
			}

			// Parse format: "owner/repo" or "https://github.com/owner/repo"
			repoStr = strings.TrimSpace(repoStr)
			if strings.HasPrefix(repoStr, "https://github.com/") {
				parts := strings.Split(strings.TrimPrefix(repoStr, "https://github.com/"), "/")
				if len(parts) >= 2 {
					return parts[0], parts[1], nil
				}
			} else if strings.Contains(repoStr, "/") {
				parts := strings.Split(repoStr, "/")
				if len(parts) == 2 {
					return parts[0], parts[1], nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("repository information not found in custom field %s", c.customField)
}

