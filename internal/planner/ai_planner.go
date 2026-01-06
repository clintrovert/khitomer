package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"

	"github.com/clintrovert/khitomer/pkg/types"
)

// AIPlanner uses OpenAI to generate implementation plans
type AIPlanner struct {
	client *openai.Client
	logger *zap.Logger
	model  string
}

// NewAIPlanner creates a new AI planner
func NewAIPlanner(apiKey, model string, logger *zap.Logger) *AIPlanner {
	client := openai.NewClient(apiKey)
	
	if model == "" {
		model = openai.GPT4TurboPreview
	}

	return &AIPlanner{
		client: client,
		logger: logger,
		model:  model,
	}
}

// Plan generates an implementation plan for a task
func (p *AIPlanner) Plan(task *types.Task) (*types.ImplementationPlan, error) {
	prompt := p.buildPrompt(task)

	resp, err := p.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an expert software engineer that creates detailed implementation plans for code changes based on Jira tickets.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.7,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	plan, err := p.parseResponse(resp.Choices[0].Message.Content, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	p.logger.Info("generated implementation plan",
		zap.String("jira_ticket", task.JiraTicketID),
		zap.Int("steps", len(plan.Steps)),
	)

	return plan, nil
}

func (p *AIPlanner) buildPrompt(task *types.Task) string {
	var sb strings.Builder
	
	sb.WriteString("Create a detailed implementation plan for the following Jira ticket:\n\n")
	sb.WriteString("**Ticket ID:** " + task.JiraTicketID + "\n")
	sb.WriteString("**Title:** " + task.Title + "\n")
	sb.WriteString("**Description:** " + task.Description + "\n")
	sb.WriteString("**Repository:** " + task.RepositoryOwner + "/" + task.RepositoryName + "\n\n")
	
	sb.WriteString("Please provide:\n")
	sb.WriteString("1. A summary of the implementation approach\n")
	sb.WriteString("2. A list of steps to complete the implementation\n")
	sb.WriteString("3. Files that need to be modified or created\n")
	sb.WriteString("4. An estimated complexity (low, medium, high)\n\n")
	
	sb.WriteString("Format your response as:\n")
	sb.WriteString("SUMMARY: <summary>\n")
	sb.WriteString("STEPS:\n")
	sb.WriteString("1. <step description> [TYPE: codegen|testing|deployment|review]\n")
	sb.WriteString("2. ...\n")
	sb.WriteString("FILES_MODIFY: <comma-separated list>\n")
	sb.WriteString("FILES_CREATE: <comma-separated list>\n")
	sb.WriteString("COMPLEXITY: <low|medium|high>\n")
	
	return sb.String()
}

func (p *AIPlanner) parseResponse(response string, task *types.Task) (*types.ImplementationPlan, error) {
	plan := &types.ImplementationPlan{
		Steps:         []types.PlanStep{},
		FilesToModify: []string{},
		FilesToCreate: []string{},
	}

	lines := strings.Split(response, "\n")
	var currentSection string
	stepOrder := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "SUMMARY:") {
			plan.Summary = strings.TrimPrefix(line, "SUMMARY:")
			plan.Summary = strings.TrimSpace(plan.Summary)
			currentSection = "summary"
		} else if strings.HasPrefix(line, "STEPS:") {
			currentSection = "steps"
		} else if strings.HasPrefix(line, "FILES_MODIFY:") {
			files := strings.TrimPrefix(line, "FILES_MODIFY:")
			files = strings.TrimSpace(files)
			if files != "" {
				plan.FilesToModify = strings.Split(files, ",")
				for i := range plan.FilesToModify {
					plan.FilesToModify[i] = strings.TrimSpace(plan.FilesToModify[i])
				}
			}
			currentSection = ""
		} else if strings.HasPrefix(line, "FILES_CREATE:") {
			files := strings.TrimPrefix(line, "FILES_CREATE:")
			files = strings.TrimSpace(files)
			if files != "" {
				plan.FilesToCreate = strings.Split(files, ",")
				for i := range plan.FilesToCreate {
					plan.FilesToCreate[i] = strings.TrimSpace(plan.FilesToCreate[i])
				}
			}
			currentSection = ""
		} else if strings.HasPrefix(line, "COMPLEXITY:") {
			plan.EstimatedComplexity = strings.TrimPrefix(line, "COMPLEXITY:")
			plan.EstimatedComplexity = strings.TrimSpace(plan.EstimatedComplexity)
			currentSection = ""
		} else if currentSection == "steps" {
			// Parse step line: "1. Description [TYPE: codegen]"
			step := p.parseStep(line, stepOrder)
			if step != nil {
				plan.Steps = append(plan.Steps, *step)
				stepOrder++
			}
		}
	}

	if plan.Summary == "" {
		plan.Summary = "Implementation plan for " + task.JiraTicketID
	}

	return plan, nil
}

func (p *AIPlanner) parseStep(line string, order int) *types.PlanStep {
	// Remove step number if present
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "."); idx != -1 {
		line = strings.TrimSpace(line[idx+1:])
	}

	// Extract type if present
	activityType := "codegen" // default
	if idx := strings.Index(line, "[TYPE:"); idx != -1 {
		typePart := line[idx:]
		if endIdx := strings.Index(typePart, "]"); endIdx != -1 {
			typeStr := strings.TrimPrefix(typePart[:endIdx+1], "[TYPE:")
			typeStr = strings.TrimSuffix(typeStr, "]")
			typeStr = strings.TrimSpace(typeStr)
			if typeStr != "" {
				activityType = typeStr
			}
			line = strings.TrimSpace(line[:idx])
		}
	}

	return &types.PlanStep{
		Order:        order,
		Description:  line,
		ActivityType: activityType,
		Parameters:   make(map[string]string),
	}
}

