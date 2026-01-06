package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/temporal/workflows"
	"github.com/clintrovert/khitomer/pkg/types"
)

// Client wraps Temporal client functionality
type Client struct {
	temporalClient client.Client
	logger         *zap.Logger
	taskQueue      string
}

// NewClient creates a new Temporal client
func NewClient(address, namespace, taskQueue string, logger *zap.Logger) (*Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  address,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	return &Client{
		temporalClient: c,
		logger:         logger,
		taskQueue:      taskQueue,
	}, nil
}

// StartWorkflow starts a new implementation workflow
func (c *Client) StartWorkflow(ctx context.Context, task *types.Task, plan *types.ImplementationPlan, repo *types.RepositoryInfo) (string, error) {
	workflowID := fmt.Sprintf("implementation-%s-%s", task.JiraTicketID, task.RepositoryName)
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: c.taskQueue,
	}

	workflowInput := workflows.WorkflowInput{
		Task:       task,
		Plan:       plan,
		Repository: repo,
	}

	we, err := c.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflows.ImplementationWorkflow, workflowInput)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	c.logger.Info("started workflow",
		zap.String("workflow_id", we.GetID()),
		zap.String("run_id", we.GetRunID()),
		zap.String("jira_ticket", task.JiraTicketID),
	)

	return we.GetID(), nil
}

// GetWorkflowStatus retrieves the status of a workflow
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (client.WorkflowRun, error) {
	workflow := c.temporalClient.GetWorkflow(ctx, workflowID, "")
	return workflow, nil
}

// CancelWorkflow cancels a running workflow
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	workflowRun := c.temporalClient.GetWorkflow(ctx, workflowID, "")
	return workflowRun.Cancel(ctx)
}

// Close closes the Temporal client
func (c *Client) Close() {
	c.temporalClient.Close()
}

