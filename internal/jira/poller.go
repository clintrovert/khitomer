package jira

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/pkg/types"
)

// Poller polls Jira for ready tasks
type Poller struct {
	client        *Client
	logger        *zap.Logger
	statusFilter  []string
	interval      time.Duration
	processedTasks map[string]bool
	mu            sync.RWMutex
}

// NewPoller creates a new Jira poller
func NewPoller(client *Client, statusFilter []string, interval time.Duration, logger *zap.Logger) *Poller {
	return &Poller{
		client:         client,
		logger:         logger,
		statusFilter:   statusFilter,
		interval:       interval,
		processedTasks: make(map[string]bool),
	}
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context, taskChan chan<- *types.Task) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Initial poll
	p.poll(ctx, taskChan)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stopping jira poller")
			return
		case <-ticker.C:
			p.poll(ctx, taskChan)
		}
	}
}

// poll performs a single poll operation
func (p *Poller) poll(ctx context.Context, taskChan chan<- *types.Task) {
	for _, status := range p.statusFilter {
		tasks, err := p.client.GetTasksByStatus(status)
		if err != nil {
			p.logger.Error("failed to get tasks by status",
				zap.String("status", status),
				zap.Error(err),
			)
			continue
		}

		for _, task := range tasks {
			if p.isProcessed(task.JiraTicketID) {
				continue
			}

			p.markProcessed(task.JiraTicketID)
			select {
			case taskChan <- task:
				p.logger.Info("found new task",
					zap.String("ticket_id", task.JiraTicketID),
					zap.String("repository", task.RepositoryName),
				)
			case <-ctx.Done():
				return
			}
		}
	}
}

// isProcessed checks if a task has been processed
func (p *Poller) isProcessed(ticketID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.processedTasks[ticketID]
}

// markProcessed marks a task as processed
func (p *Poller) markProcessed(ticketID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processedTasks[ticketID] = true
}

// ClearProcessed clears the processed tasks list (for testing)
func (p *Poller) ClearProcessed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processedTasks = make(map[string]bool)
}

