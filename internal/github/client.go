package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/pkg/types"
)

// Client wraps GitHub API and Git operations
type Client struct {
	apiClient    *github.Client
	logger       *zap.Logger
	accessToken  string
	workspaceDir string
}

// NewClient creates a new GitHub client
func NewClient(accessToken, workspaceDir string, logger *zap.Logger) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		apiClient:    github.NewClient(tc),
		logger:       logger,
		accessToken:  accessToken,
		workspaceDir: workspaceDir,
	}
}

// CloneRepository clones a GitHub repository to the workspace
func (c *Client) CloneRepository(ctx context.Context, owner, repo, branch string) (string, error) {
	repoPath := filepath.Join(c.workspaceDir, owner, repo)
	
	// Remove existing directory if it exists
	if _, err := os.Stat(repoPath); err == nil {
		os.RemoveAll(repoPath)
	}

	// Create directory
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	cloneURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", c.accessToken, owner, repo)
	
	_, err := git.PlainCloneContext(ctx, repoPath, false, &git.CloneOptions{
		URL:           cloneURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Progress:      os.Stdout,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	c.logger.Info("cloned repository",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("path", repoPath),
	)

	return repoPath, nil
}

// GetRepositoryPath returns the path to a cloned repository
func (c *Client) GetRepositoryPath(owner, repo string) string {
	return filepath.Join(c.workspaceDir, owner, repo)
}

// CreateBranch creates a new branch from the base branch
func (c *Client) CreateBranch(repoPath, baseBranch, newBranch string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Checkout base branch
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(baseBranch),
	})
	if err != nil {
		return fmt.Errorf("failed to checkout base branch: %w", err)
	}

	// Create and checkout new branch
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(newBranch),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	c.logger.Info("created branch",
		zap.String("branch", newBranch),
		zap.String("repo_path", repoPath),
	)

	return nil
}

// CommitChanges commits changes to the repository
func (c *Client) CommitChanges(repoPath, message string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &git.Signature{
			Name:  "Khitomer Bot",
			Email: "khitomer@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	c.logger.Info("committed changes",
		zap.String("message", message),
		zap.String("repo_path", repoPath),
	)

	return nil
}

// PushBranch pushes a branch to GitHub
func (c *Client) PushBranch(ctx context.Context, repoPath, branch string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	remote, err := r.Remote("origin")
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	err = remote.PushContext(ctx, &git.PushOptions{
		RefSpecs: []git.RefSpec{git.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch))},
		Auth:     nil, // Will use token from URL
	})
	if err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	c.logger.Info("pushed branch",
		zap.String("branch", branch),
		zap.String("repo_path", repoPath),
	)

	return nil
}

// CreatePullRequest creates a pull request
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, baseBranch, headBranch, title, body string) (*types.PRInfo, error) {
	newPR := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(headBranch),
		Base:  github.String(baseBranch),
		Body:  github.String(body),
	}

	pr, _, err := c.apiClient.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	prInfo := &types.PRInfo{
		PRNumber:    int64(pr.GetNumber()),
		PRURL:       pr.GetHTMLURL(),
		Title:       pr.GetTitle(),
		Description: pr.GetBody(),
		Status:      pr.GetState(),
	}

	c.logger.Info("created pull request",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int64("pr_number", prInfo.PRNumber),
		zap.String("pr_url", prInfo.PRURL),
	)

	return prInfo, nil
}

