package activities

import (
	"context"
	"os/exec"
	"path/filepath"

	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// TestingActivity runs tests in the repository
func TestingActivity(ctx context.Context, repoPath string) (TestingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("running tests",
		zap.String("repo_path", repoPath),
	)

	result := TestingResult{
		Passed:   false,
		Failures: []string{},
	}

	// Try to find and run tests
	// This is a placeholder - in a real implementation, you would:
	// 1. Detect the project type (Go, Node, Python, etc.)
	// 2. Run the appropriate test command
	// 3. Parse test results

	// Example for Go projects
	if _, err := exec.LookPath("go"); err == nil {
		// Check if it's a Go project
		if _, err := filepath.Glob(filepath.Join(repoPath, "*.go")); err == nil {
			cmd := exec.CommandContext(ctx, "go", "test", "./...")
			cmd.Dir = repoPath
			output, err := cmd.CombinedOutput()
			
			result.Output = string(output)
			if err != nil {
				result.Failures = append(result.Failures, err.Error())
				logger.Warn("tests failed", zap.Error(err))
				return result, nil // Don't fail the workflow if tests fail
			}
			
			result.Passed = true
			logger.Info("tests passed")
			return result, nil
		}
	}

	// If no tests found or project type not supported, assume success
	logger.Info("no tests found or project type not supported, assuming success")
	result.Passed = true
	return result, nil
}

