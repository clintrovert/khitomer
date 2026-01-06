package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/github"
	"github.com/clintrovert/khitomer/pkg/types"
)

// GitHubActivities handles GitHub-related activities
type GitHubActivities struct {
	githubClient *github.Client
	logger       *zap.Logger
}

// NewGitHubActivities creates a new GitHub activities handler
func NewGitHubActivities(githubClient *github.Client, logger *zap.Logger) *GitHubActivities {
	return &GitHubActivities{
		githubClient: githubClient,
		logger:       logger,
	}
}

// CloneRepositoryActivity clones a GitHub repository
func (a *GitHubActivities) CloneRepositoryActivity(ctx context.Context, repo *types.RepositoryInfo) (GitHubOperationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("cloning repository",
		zap.String("owner", repo.Owner),
		zap.String("name", repo.Name),
	)

	repoPath, err := a.githubClient.CloneRepository(ctx, repo.Owner, repo.Name, repo.BaseBranch)
	if err != nil {
		return GitHubOperationResult{Success: false, Message: err.Error()}, err
	}

	return GitHubOperationResult{
		Success:        true,
		Message:        "repository cloned successfully",
		RepositoryPath: repoPath,
	}, nil
}

// CreateBranchActivity creates a new branch
func (a *GitHubActivities) CreateBranchActivity(ctx context.Context, repo *types.RepositoryInfo, branchName string) (GitHubOperationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("creating branch",
		zap.String("branch", branchName),
		zap.String("repo", repo.Name),
	)

	// Get repository path
	repoPath := a.githubClient.GetRepositoryPath(repo.Owner, repo.Name)

	err := a.githubClient.CreateBranch(repoPath, repo.BaseBranch, branchName)
	if err != nil {
		return GitHubOperationResult{Success: false, Message: err.Error()}, err
	}

	return GitHubOperationResult{
		Success:    true,
		Message:    "branch created successfully",
		BranchName: branchName,
	}, nil
}

// CommitChangesActivity commits changes to the repository
func (a *GitHubActivities) CommitChangesActivity(ctx context.Context, repo *types.RepositoryInfo, repoPath, message string) (GitHubOperationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("committing changes",
		zap.String("repo", repo.Name),
		zap.String("message", message),
	)

	err := a.githubClient.CommitChanges(repoPath, message)
	if err != nil {
		return GitHubOperationResult{Success: false, Message: err.Error()}, err
	}

	// Push the branch
	err = a.githubClient.PushBranch(ctx, repoPath, repo.FeatureBranch)
	if err != nil {
		return GitHubOperationResult{Success: false, Message: err.Error()}, err
	}

	return GitHubOperationResult{
		Success: true,
		Message: "changes committed and pushed successfully",
	}, nil
}

// CreatePRActivity creates a pull request
func (a *GitHubActivities) CreatePRActivity(ctx context.Context, repo *types.RepositoryInfo, title, description string) (GitHubOperationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("creating pull request",
		zap.String("repo", repo.Name),
		zap.String("title", title),
	)

	prInfo, err := a.githubClient.CreatePullRequest(ctx, repo.Owner, repo.Name, repo.BaseBranch, repo.FeatureBranch, title, description)
	if err != nil {
		return GitHubOperationResult{Success: false, Message: err.Error()}, err
	}

	return GitHubOperationResult{
		Success: true,
		Message: "pull request created successfully",
		PRInfo:   prInfo,
	}, nil
}

