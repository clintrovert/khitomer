package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/internal/temporal"
	"github.com/clintrovert/khitomer/pkg/types"
)

// Handler handles REST API requests
type Handler struct {
	temporalClient *temporal.Client
	logger         *zap.Logger
}

// NewHandler creates a new REST handler
func NewHandler(temporalClient *temporal.Client, logger *zap.Logger) *Handler {
	return &Handler{
		temporalClient: temporalClient,
		logger:          logger,
	}
}

// StartWorkflowRequest represents a request to start a workflow
type StartWorkflowRequest struct {
	JiraTicketID   string `json:"jira_ticket_id"`
	RepositoryOwner string `json:"repository_owner"`
	RepositoryName  string `json:"repository_name"`
	BaseBranch     string `json:"base_branch"`
}

// StartWorkflowResponse represents the response from starting a workflow
type StartWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
}

// GetWorkflowStatusResponse represents the workflow status
type GetWorkflowStatusResponse struct {
	WorkflowID   string `json:"workflow_id"`
	Status       string `json:"status"`
	JiraTicketID string `json:"jira_ticket_id,omitempty"`
	PRURL        string `json:"pr_url,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// StartWorkflow handles POST /workflows
func (h *Handler) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	var req StartWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create task from request
	task := &types.Task{
		JiraTicketID:     req.JiraTicketID,
		RepositoryOwner: req.RepositoryOwner,
		RepositoryName:  req.RepositoryName,
		BaseBranch:      req.BaseBranch,
	}

	// Create minimal plan (for manual triggers, planning would be done separately)
	plan := &types.ImplementationPlan{
		Summary: "Manual workflow trigger",
		Steps:   []types.PlanStep{},
	}

	repo := &types.RepositoryInfo{
		Owner:      req.RepositoryOwner,
		Name:       req.RepositoryName,
		BaseBranch: req.BaseBranch,
		CloneURL:   "https://github.com/" + req.RepositoryOwner + "/" + req.RepositoryName,
	}

	workflowID, err := h.temporalClient.StartWorkflow(r.Context(), task, plan, repo)
	if err != nil {
		h.logger.Error("failed to start workflow", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := StartWorkflowResponse{
		WorkflowID: workflowID,
		Status:     "started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetWorkflowStatus handles GET /workflows/{id}
func (h *Handler) GetWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")

	_, err := h.temporalClient.GetWorkflowStatus(r.Context(), workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := GetWorkflowStatusResponse{
		WorkflowID: workflowID,
		Status:     "running", // TODO: Get actual status from workflow
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CancelWorkflow handles DELETE /workflows/{id}
func (h *Handler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")

	err := h.temporalClient.CancelWorkflow(r.Context(), workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// RegisterRoutes registers REST API routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/workflows", h.StartWorkflow)
	r.Get("/workflows/{id}", h.GetWorkflowStatus)
	r.Delete("/workflows/{id}", h.CancelWorkflow)
}

