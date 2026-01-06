package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/activities"
	"github.com/clintrovert/khitomer/pkg/types"
)

// ImplementationWorkflow orchestrates the implementation of a Jira task
func ImplementationWorkflow(ctx workflow.Context, input WorkflowInput) (*types.PRInfo, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting implementation workflow",
		zap.String("jira_ticket", input.Task.JiraTicketID),
		zap.String("repository", input.Repository.Name),
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Clone repository
	var cloneResult activities.GitHubOperationResult
	err := workflow.ExecuteActivity(ctx, activities.CloneRepositoryActivity, input.Repository).Get(ctx, &cloneResult)
	if err != nil {
		logger.Error("failed to clone repository", zap.Error(err))
		return nil, err
	}

	// Step 2: Create feature branch
	var branchResult activities.GitHubOperationResult
	branchName := generateBranchName(input.Task.JiraTicketID, input.Task.Title)
	err = workflow.ExecuteActivity(ctx, activities.CreateBranchActivity, input.Repository, branchName).Get(ctx, &branchResult)
	if err != nil {
		logger.Error("failed to create branch", zap.Error(err))
		return nil, err
	}
	input.Repository.FeatureBranch = branchResult.BranchName

	// Step 3: Generate/modify code
	var codegenResult activities.CodeGenerationResult
	err = workflow.ExecuteActivity(ctx, activities.CodeGenerationActivity, input.Task, input.Plan, cloneResult.RepositoryPath).Get(ctx, &codegenResult)
	if err != nil {
		logger.Error("failed to generate code", zap.Error(err))
		return nil, err
	}

	// Step 4: Run tests
	var testResult activities.TestingResult
	err = workflow.ExecuteActivity(ctx, activities.TestingActivity, cloneResult.RepositoryPath).Get(ctx, &testResult)
	if err != nil {
		logger.Error("tests failed", zap.Error(err))
		// Continue even if tests fail - let humans review
	}

	// Step 5: Commit changes
	var commitResult activities.GitHubOperationResult
	err = workflow.ExecuteActivity(ctx, activities.CommitChangesActivity, input.Repository, cloneResult.RepositoryPath, codegenResult.Summary).Get(ctx, &commitResult)
	if err != nil {
		logger.Error("failed to commit changes", zap.Error(err))
		return nil, err
	}

	// Step 6: Create PR
	var prResult activities.GitHubOperationResult
	prTitle := generatePRTitle(input.Task.JiraTicketID, input.Task.Title)
	prDescription := generatePRDescription(input.Task, input.Plan)
	err = workflow.ExecuteActivity(ctx, activities.CreatePRActivity, input.Repository, prTitle, prDescription).Get(ctx, &prResult)
	if err != nil {
		logger.Error("failed to create PR", zap.Error(err))
		return nil, err
	}

	// Step 7: Update Jira with PR link
	var jiraResult activities.JiraUpdateResult
	err = workflow.ExecuteActivity(ctx, activities.UpdateJiraActivity, input.Task.JiraTicketID, prResult.PRInfo.PRURL).Get(ctx, &jiraResult)
	if err != nil {
		logger.Error("failed to update Jira", zap.Error(err))
		// Non-fatal - PR was created successfully
	}

	logger.Info("implementation workflow completed",
		zap.String("pr_url", prResult.PRInfo.PRURL),
	)

	return prResult.PRInfo, nil
}

func generateBranchName(ticketID, title string) string {
	// Simple branch name: khitomer/JIRA-123-short-title
	shortTitle := truncateString(title, 30)
	return "khitomer/" + ticketID + "-" + sanitizeBranchName(shortTitle)
}

func generatePRTitle(ticketID, title string) string {
	return ticketID + ": " + title
}

func generatePRDescription(task *types.Task, plan *types.ImplementationPlan) string {
	desc := "## Implementation for " + task.JiraTicketID + "\n\n"
	desc += "**Jira Ticket:** " + task.JiraTicketID + "\n"
	desc += "**Description:** " + task.Description + "\n\n"
	desc += "## Implementation Plan\n\n"
	desc += plan.Summary + "\n\n"
	desc += "## Steps\n\n"
	for i, step := range plan.Steps {
		desc += fmt.Sprintf("%d. %s\n", i+1, step.Description)
	}
	return desc
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func sanitizeBranchName(s string) string {
	// Remove special characters and replace spaces with hyphens
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		} else if r == ' ' {
			result += "-"
		}
	}
	return result
}
