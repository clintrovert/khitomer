package leader

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/jira"
	"github.com/clintrovert/khitomer/internal/planner"
	"github.com/clintrovert/khitomer/internal/temporal"
	"github.com/clintrovert/khitomer/pkg/types"
)

// Orchestrator coordinates Jira polling, AI planning, and workflow spawning
type Orchestrator struct {
	jiraPoller  *jira.Poller
	planner     planner.Planner
	temporalClient *temporal.Client
	logger      *zap.Logger
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(
	jiraPoller *jira.Poller,
	planner planner.Planner,
	temporalClient *temporal.Client,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		jiraPoller:     jiraPoller,
		planner:        planner,
		temporalClient: temporalClient,
		logger:         logger,
	}
}

// Start starts the orchestration loop
func (o *Orchestrator) Start(ctx context.Context) error {
	taskChan := make(chan *types.Task, 10)

	// Start Jira polling in background
	go o.jiraPoller.Start(ctx, taskChan)

	// Process tasks as they come in
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case task := <-taskChan:
			if err := o.processTask(ctx, task); err != nil {
				o.logger.Error("failed to process task",
					zap.String("jira_ticket", task.JiraTicketID),
					zap.Error(err),
				)
			}
		}
	}
}

// processTask processes a single task
func (o *Orchestrator) processTask(ctx context.Context, task *types.Task) error {
	o.logger.Info("processing task",
		zap.String("jira_ticket", task.JiraTicketID),
		zap.String("repository", task.RepositoryName),
	)

	// Generate implementation plan
	plan, err := o.planner.Plan(task)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// Create repository info
	repo := &types.RepositoryInfo{
		Owner:      task.RepositoryOwner,
		Name:       task.RepositoryName,
		BaseBranch: task.BaseBranch,
		CloneURL:   task.RepositoryURL,
	}

	// Start workflow
	workflowID, err := o.temporalClient.StartWorkflow(ctx, task, plan, repo)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	o.logger.Info("started workflow for task",
		zap.String("jira_ticket", task.JiraTicketID),
		zap.String("workflow_id", workflowID),
	)

	return nil
}

