package activities

import (
	"context"

	"github.com/clintrovert/khitomer/pkg/types"
)

// Activity functions that will be registered with Temporal worker
// These are wrapper functions that call the actual activity implementations

var (
	githubActivities *GitHubActivities
	jiraActivities   *JiraActivities
)

// SetGitHubActivities sets the GitHub activities implementation
func SetGitHubActivities(ga *GitHubActivities) {
	githubActivities = ga
}

// SetJiraActivities sets the Jira activities implementation
func SetJiraActivities(ja *JiraActivities) {
	jiraActivities = ja
}

// CloneRepositoryActivity is the activity function for cloning repositories
func CloneRepositoryActivity(ctx context.Context, repo *types.RepositoryInfo) (GitHubOperationResult, error) {
	if githubActivities == nil {
		return GitHubOperationResult{Success: false, Message: "GitHub activities not initialized"}, nil
	}
	return githubActivities.CloneRepositoryActivity(ctx, repo)
}

// CreateBranchActivity is the activity function for creating branches
func CreateBranchActivity(ctx context.Context, repo *types.RepositoryInfo, branchName string) (GitHubOperationResult, error) {
	if githubActivities == nil {
		return GitHubOperationResult{Success: false, Message: "GitHub activities not initialized"}, nil
	}
	return githubActivities.CreateBranchActivity(ctx, repo, branchName)
}

// CommitChangesActivity is the activity function for committing changes
func CommitChangesActivity(ctx context.Context, repo *types.RepositoryInfo, repoPath, message string) (GitHubOperationResult, error) {
	if githubActivities == nil {
		return GitHubOperationResult{Success: false, Message: "GitHub activities not initialized"}, nil
	}
	return githubActivities.CommitChangesActivity(ctx, repo, repoPath, message)
}

// CreatePRActivity is the activity function for creating pull requests
func CreatePRActivity(ctx context.Context, repo *types.RepositoryInfo, title, description string) (GitHubOperationResult, error) {
	if githubActivities == nil {
		return GitHubOperationResult{Success: false, Message: "GitHub activities not initialized"}, nil
	}
	return githubActivities.CreatePRActivity(ctx, repo, title, description)
}

// UpdateJiraActivity is the activity function for updating Jira
func UpdateJiraActivity(ctx context.Context, ticketID, prURL string) (JiraUpdateResult, error) {
	if jiraActivities == nil {
		return JiraUpdateResult{Success: false, Message: "Jira activities not initialized"}, nil
	}
	return jiraActivities.UpdateJiraActivity(ctx, ticketID, prURL)
}

