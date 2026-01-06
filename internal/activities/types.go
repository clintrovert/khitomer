package activities

import (
	"github.com/clintrovert/khitomer/pkg/types"
)

// GitHubOperationResult contains the result of a GitHub operation
type GitHubOperationResult struct {
	Success       bool
	Message       string
	PRInfo        *types.PRInfo
	BranchName    string
	RepositoryPath string
}

// CodeGenerationResult contains the result of code generation
type CodeGenerationResult struct {
	Success       bool
	ModifiedFiles []string
	CreatedFiles  []string
	Summary       string
}

// TestingResult contains the result of testing
type TestingResult struct {
	Passed   bool
	Output   string
	Failures []string
}

// JiraUpdateResult contains the result of a Jira update
type JiraUpdateResult struct {
	Success bool
	Message string
}

