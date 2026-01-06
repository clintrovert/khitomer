package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/clintrovert/khitomer/internal/temporal"
	"github.com/clintrovert/khitomer/pkg/types"
	pb "github.com/clintrovert/khitomer/proto"
)

// Server implements the LeaderService gRPC service
type Server struct {
	pb.UnimplementedLeaderServiceServer
	temporalClient *temporal.Client
	logger         *zap.Logger
}

// NewServer creates a new gRPC server
func NewServer(temporalClient *temporal.Client, logger *zap.Logger) *Server {
	return &Server{
		temporalClient: temporalClient,
		logger:          logger,
	}
}

// Register registers the server with a gRPC server
func (s *Server) Register(grpcServer *grpc.Server) {
	pb.RegisterLeaderServiceServer(grpcServer, s)
}

// StartWorkflow starts a workflow
func (s *Server) StartWorkflow(ctx context.Context, req *pb.StartWorkflowRequest) (*pb.StartWorkflowResponse, error) {
	task := &types.Task{
		JiraTicketID:     req.JiraTicketId,
		RepositoryOwner: req.RepositoryOwner,
		RepositoryName:  req.RepositoryName,
		BaseBranch:      req.BaseBranch,
	}

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

	workflowID, err := s.temporalClient.StartWorkflow(ctx, task, plan, repo)
	if err != nil {
		return nil, err
	}

	return &pb.StartWorkflowResponse{
		WorkflowId: workflowID,
		Status:     "started",
	}, nil
}

// GetWorkflowStatus gets the status of a workflow
func (s *Server) GetWorkflowStatus(ctx context.Context, req *pb.GetWorkflowStatusRequest) (*pb.GetWorkflowStatusResponse, error) {
	_, err := s.temporalClient.GetWorkflowStatus(ctx, req.WorkflowId)
	if err != nil {
		return &pb.GetWorkflowStatusResponse{
			WorkflowId:   req.WorkflowId,
			Status:       "not_found",
			ErrorMessage: err.Error(),
		}, nil
	}

	// TODO: Get actual status from workflow
	return &pb.GetWorkflowStatusResponse{
		WorkflowId: req.WorkflowId,
		Status:     "running",
	}, nil
}

// CancelWorkflow cancels a workflow
func (s *Server) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*pb.CancelWorkflowResponse, error) {
	err := s.temporalClient.CancelWorkflow(ctx, req.WorkflowId)
	if err != nil {
		return &pb.CancelWorkflowResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.CancelWorkflowResponse{
		Success: true,
		Message: "workflow cancelled",
	}, nil
}

// GetProcessedTasks gets a list of processed tasks
func (s *Server) GetProcessedTasks(ctx context.Context, req *pb.GetProcessedTasksRequest) (*pb.GetProcessedTasksResponse, error) {
	// TODO: Implement task tracking
	return &pb.GetProcessedTasksResponse{
		Tasks: []*pb.ProcessedTask{},
		Total: 0,
	}, nil
}

