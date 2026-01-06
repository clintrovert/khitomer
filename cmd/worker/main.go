package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/activities"
	"github.com/clintrovert/khitomer/internal/github"
	"github.com/clintrovert/khitomer/internal/jira"
	workflows "github.com/clintrovert/khitomer/internal/temporal/workflows"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Get configuration from environment
	temporalAddress := getEnv("TEMPORAL_ADDRESS", "localhost:7233")
	temporalNamespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TASK_QUEUE", "implementation-queue")
	githubToken := getEnv("GITHUB_TOKEN", "")
	workspaceDir := getEnv("WORKSPACE_DIR", "/tmp/khitomer-workspace")
	jiraBaseURL := getEnv("JIRA_BASE_URL", "")
	jiraUsername := getEnv("JIRA_USERNAME", "")
	jiraToken := getEnv("JIRA_TOKEN", "")

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: temporalNamespace,
	})
	if err != nil {
		logger.Fatal("failed to create temporal client", zap.Error(err))
	}
	defer c.Close()

	// Create GitHub client
	githubClient := github.NewClient(githubToken, workspaceDir, logger)
	
	// Create Jira client (for updating Jira)
	var jiraClient *jira.Client
	if jiraBaseURL != "" && jiraUsername != "" && jiraToken != "" {
		jiraClient, err = jira.NewClient(jiraBaseURL, jiraUsername, jiraToken, "", "", logger)
		if err != nil {
			logger.Warn("failed to create jira client", zap.Error(err))
		}
	}

	// Initialize activities
	githubActivities := activities.NewGitHubActivities(githubClient, logger)
	activities.SetGitHubActivities(githubActivities)

	if jiraClient != nil {
		jiraActivities := activities.NewJiraActivities(jiraClient, logger)
		activities.SetJiraActivities(jiraActivities)
	}

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflow
	w.RegisterWorkflow(workflows.ImplementationWorkflow)

	// Register activities
	w.RegisterActivity(activities.CloneRepositoryActivity)
	w.RegisterActivity(activities.CreateBranchActivity)
	w.RegisterActivity(activities.CodeGenerationActivity)
	w.RegisterActivity(activities.TestingActivity)
	w.RegisterActivity(activities.CommitChangesActivity)
	w.RegisterActivity(activities.CreatePRActivity)
	w.RegisterActivity(activities.UpdateJiraActivity)

	// Start worker
	logger.Info("starting worker",
		zap.String("task_queue", taskQueue),
		zap.String("namespace", temporalNamespace),
	)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal("worker failed", zap.Error(err))
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down worker")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

