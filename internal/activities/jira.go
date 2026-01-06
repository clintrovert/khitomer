package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/jira"
)

// JiraActivities handles Jira-related activities
type JiraActivities struct {
	jiraClient *jira.Client
	logger     *zap.Logger
}

// NewJiraActivities creates a new Jira activities handler
func NewJiraActivities(jiraClient *jira.Client, logger *zap.Logger) *JiraActivities {
	return &JiraActivities{
		jiraClient: jiraClient,
		logger:     logger,
	}
}

// UpdateJiraActivity updates Jira with PR link and status
func (a *JiraActivities) UpdateJiraActivity(ctx context.Context, ticketID, prURL string) (JiraUpdateResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("updating Jira",
		zap.String("ticket_id", ticketID),
		zap.String("pr_url", prURL),
	)

	comment := fmt.Sprintf("Pull request created: %s", prURL)
	err := a.jiraClient.AddComment(ticketID, comment)
	if err != nil {
		logger.Error("failed to add comment", zap.Error(err))
		return JiraUpdateResult{Success: false, Message: err.Error()}, err
	}

	// Optionally update status to "In Review" or similar
	// err = a.jiraClient.UpdateTaskStatus(ticketID, "In Review")
	// if err != nil {
	// 	logger.Warn("failed to update status", zap.Error(err))
	// }

	return JiraUpdateResult{
		Success: true,
		Message: "Jira updated successfully",
	}, nil
}

