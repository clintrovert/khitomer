package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	grpcapi "github.com/clintrovert/khitomer/internal/api/grpc"
	"github.com/clintrovert/khitomer/internal/api/rest"
	"github.com/clintrovert/khitomer/internal/jira"
	"github.com/clintrovert/khitomer/internal/leader"
	"github.com/clintrovert/khitomer/internal/planner"
	"github.com/clintrovert/khitomer/internal/temporal"
	pb "github.com/clintrovert/khitomer/proto"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync()

	// Get configuration from environment
	temporalAddress := getEnv("TEMPORAL_ADDRESS", "localhost:7233")
	temporalNamespace := getEnv("TEMPORAL_NAMESPACE", "default")
	taskQueue := getEnv("TASK_QUEUE", "implementation-queue")
	jiraBaseURL := getEnv("JIRA_BASE_URL", "")
	jiraUsername := getEnv("JIRA_USERNAME", "")
	jiraToken := getEnv("JIRA_TOKEN", "")
	jiraProjectKey := getEnv("JIRA_PROJECT_KEY", "")
	jiraCustomField := getEnv("JIRA_CUSTOM_FIELD", "Repository")
	jiraStatusFilter := getEnv("JIRA_STATUS_FILTER", "Ready for Development")
	jiraPollInterval := getEnv("JIRA_POLL_INTERVAL", "5m")
	openaiAPIKey := getEnv("OPENAI_API_KEY", "")
	openaiModel := getEnv("OPENAI_MODEL", "")
	restPort := getEnv("REST_PORT", "8080")
	grpcPort := getEnv("GRPC_PORT", "9090")

	// Parse poll interval
	pollInterval, err := time.ParseDuration(jiraPollInterval)
	if err != nil {
		logger.Warn("invalid poll interval, using default", zap.Error(err))
		pollInterval = 5 * time.Minute
	}

	// Create Temporal client
	temporalClient, err := temporal.NewClient(temporalAddress, temporalNamespace, taskQueue, logger)
	if err != nil {
		logger.Fatal("failed to create temporal client", zap.Error(err))
	}
	defer temporalClient.Close()

	// Create Jira client
	jiraClient, err := jira.NewClient(jiraBaseURL, jiraUsername, jiraToken, jiraProjectKey, jiraCustomField, logger)
	if err != nil {
		logger.Fatal("failed to create jira client", zap.Error(err))
	}

	// Create Jira poller
	statusFilter := []string{jiraStatusFilter}
	jiraPoller := jira.NewPoller(jiraClient, statusFilter, pollInterval, logger)

	// Create AI planner
	aiPlanner := planner.NewAIPlanner(openaiAPIKey, openaiModel, logger)

	// Create orchestrator
	orchestrator := leader.NewOrchestrator(jiraPoller, aiPlanner, temporalClient, logger)

	// Create REST API handler
	restHandler := rest.NewHandler(temporalClient, logger)

	// Create gRPC server
	grpcServer := grpcapi.NewServer(temporalClient, logger)

	// Setup REST API
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		restHandler.RegisterRoutes(r)
	})
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Start REST server
	restAddr := fmt.Sprintf(":%s", restPort)
	restServer := &http.Server{
		Addr:    restAddr,
		Handler: router,
	}

	go func() {
		logger.Info("starting REST API server", zap.String("address", restAddr))
		if err := restServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start REST server", zap.Error(err))
		}
	}()

	// Start gRPC server
	grpcAddr := fmt.Sprintf(":%s", grpcPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Error(err))
	}

	grpcSrv := grpc.NewServer()
	grpcServer.Register(grpcSrv)
	pb.RegisterLeaderServiceServer(grpcSrv, grpcServer)

	go func() {
		logger.Info("starting gRPC server", zap.String("address", grpcAddr))
		if err := grpcSrv.Serve(grpcListener); err != nil {
			logger.Fatal("failed to start gRPC server", zap.Error(err))
		}
	}()

	// Start orchestrator
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := orchestrator.Start(ctx); err != nil {
			logger.Error("orchestrator failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down")

	// Shutdown orchestrator
	cancel()

	// Shutdown servers
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	restServer.Shutdown(shutdownCtx)
	grpcSrv.GracefulStop()

	logger.Info("shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
