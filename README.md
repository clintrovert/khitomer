# Khitomer

Khitomer is an AI orchestration platform that integrates with Jira and GitHub to automatically plan and implement tasks. It uses Temporal for workflow orchestration, polls Jira for ready tasks, generates AI-powered implementation plans, and creates GitHub pull requests for human review.

## Architecture

Khitomer consists of two main services:

- **Leader Service**: Polls Jira for ready tasks, uses AI to generate implementation plans, and spawns Temporal workflows
- **Worker Service**: Executes workflows by cloning repositories, generating code, running tests, and creating pull requests

### Data Flow

1. Leader polls Jira for tasks with a specific status
2. Leader extracts repository information from Jira custom field
3. Leader uses AI/LLM to analyze task and generate implementation plan
4. Leader starts Temporal workflow with plan and task metadata
5. Workers poll Temporal for work and execute activities
6. Workers clone GitHub repository, create branch, generate/modify code
7. Workers run tests, commit changes, and create PR
8. Workers update Jira with PR link and progress

## Features

- **Jira Integration**: Automatic polling for ready tasks with configurable status filters
- **AI Planning**: Uses OpenAI (or compatible LLM) to generate detailed implementation plans
- **GitHub Operations**: Automated repository cloning, branching, committing, and PR creation
- **Temporal Orchestration**: Reliable workflow execution with retries and error handling
- **REST & gRPC APIs**: Manual workflow triggers and status monitoring
- **Docker Support**: Easy deployment with docker-compose

## Prerequisites

- Go 1.24 or later
- Docker and docker-compose (for local Temporal server)
- Jira instance with API access
- GitHub account with repository access
- OpenAI API key (or compatible LLM service)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/clintrovert/khitomer.git
cd khitomer
```

### 2. Configure Environment

Copy the example environment file and fill in your credentials:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
# Temporal Configuration
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default
TASK_QUEUE=implementation-queue

# Jira Configuration
JIRA_BASE_URL=https://your-jira-instance.atlassian.net
JIRA_USERNAME=your-email@example.com
JIRA_TOKEN=your-api-token
JIRA_PROJECT_KEY=PROJ
JIRA_CUSTOM_FIELD=Repository
JIRA_STATUS_FILTER=Ready for Development
JIRA_POLL_INTERVAL=5m

# GitHub Configuration
GITHUB_TOKEN=your-github-token
WORKSPACE_DIR=/tmp/khitomer-workspace

# AI/LLM Configuration
OPENAI_API_KEY=your-openai-api-key
OPENAI_MODEL=gpt-4-turbo-preview

# API Server Configuration
REST_PORT=8080
GRPC_PORT=9090
```

### 3. Start Temporal Server

```bash
docker-compose up -d temporal postgresql
```

Wait for Temporal to be ready (check logs: `docker-compose logs temporal`)

### 4. Build and Run Services

#### Option A: Run Locally

```bash
# Terminal 1: Start Leader Service
go run cmd/leader/main.go

# Terminal 2: Start Worker Service
go run cmd/worker/main.go
```

#### Option B: Use Docker

```bash
docker-compose up leader worker
```

## Configuration

### Jira Setup

1. Create a custom field in Jira (e.g., "Repository") to store GitHub repository references
2. Format: `owner/repo` or full URL `https://github.com/owner/repo`
3. Set the custom field name in `JIRA_CUSTOM_FIELD` environment variable
4. Configure which Jira status indicates "ready" tasks in `JIRA_STATUS_FILTER`

### GitHub Setup

1. Generate a GitHub Personal Access Token with `repo` permissions
2. Set the token in `GITHUB_TOKEN` environment variable
3. Ensure the token has access to all repositories you want to work with

### AI/LLM Configuration

Khitomer uses OpenAI by default, but can be adapted for other LLM providers:

- **OpenAI**: Set `OPENAI_API_KEY` and optionally `OPENAI_MODEL`
- **Other Providers**: Modify `internal/planner/ai_planner.go` to use different clients

## API Endpoints

### REST API

- `POST /api/v1/workflows` - Manually trigger a workflow
  ```json
  {
    "jira_ticket_id": "PROJ-123",
    "repository_owner": "owner",
    "repository_name": "repo",
    "base_branch": "main"
  }
  ```

- `GET /api/v1/workflows/{id}` - Get workflow status
- `DELETE /api/v1/workflows/{id}` - Cancel a workflow
- `GET /health` - Health check

### gRPC API

The gRPC service implements the `LeaderService` defined in `proto/leader.proto`:

- `StartWorkflow` - Start a workflow manually
- `GetWorkflowStatus` - Get workflow status
- `CancelWorkflow` - Cancel a running workflow
- `GetProcessedTasks` - List processed tasks

## Project Structure

```
khitomer/
├── cmd/
│   ├── leader/          # Leader service entry point
│   └── worker/          # Worker service entry point
├── internal/
│   ├── api/             # REST and gRPC API handlers
│   ├── jira/            # Jira client and polling logic
│   ├── github/          # GitHub API client and operations
│   ├── planner/         # AI-based planning service
│   ├── leader/           # Orchestration logic
│   ├── temporal/        # Temporal client and workflow definitions
│   └── activities/      # Activity implementations
├── proto/               # Protocol buffer definitions
├── pkg/types/           # Shared types
├── Dockerfile.leader    # Leader service Dockerfile
├── Dockerfile.worker    # Worker service Dockerfile
└── docker-compose.yml   # Docker Compose configuration
```

## Development

### Generate Protocol Buffers

```bash
make proto
```

### Build Binaries

```bash
make build
```

### Run Tests

```bash
go test ./...
```

### Code Generation

The project uses Protocol Buffers for gRPC. After modifying `.proto` files:

```bash
make proto
```

## Workflow Details

The main workflow (`ImplementationWorkflow`) executes the following steps:

1. **Clone Repository**: Clone the GitHub repository to local workspace
2. **Create Branch**: Create a feature branch (format: `khitomer/JIRA-123-description`)
3. **Generate Code**: Generate or modify code based on the AI plan
4. **Run Tests**: Execute tests in the repository
5. **Commit Changes**: Commit changes to the feature branch
6. **Create PR**: Create a pull request for human review
7. **Update Jira**: Add PR link as a comment in the Jira ticket

## Troubleshooting

### Temporal Connection Issues

- Ensure Temporal server is running: `docker-compose ps`
- Check Temporal logs: `docker-compose logs temporal`
- Verify `TEMPORAL_ADDRESS` matches your Temporal server

### Jira Authentication Errors

- Verify `JIRA_TOKEN` is a valid API token (not password)
- Check `JIRA_BASE_URL` format (should include protocol: `https://`)
- Ensure user has access to the project specified in `JIRA_PROJECT_KEY`

### GitHub Permission Errors

- Verify GitHub token has `repo` scope
- Check token hasn't expired
- Ensure token has access to the repository

### AI Planning Failures

- Verify `OPENAI_API_KEY` is valid
- Check API quota/rate limits
- Review logs for specific error messages

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license here]

## Support

For issues and questions, please open an issue on GitHub.
