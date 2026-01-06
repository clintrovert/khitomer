package workflows

import (
	"github.com/clintrovert/khitomer/pkg/types"
)

// WorkflowInput is the input for the implementation workflow
type WorkflowInput struct {
	Task       *types.Task
	Plan       *types.ImplementationPlan
	Repository *types.RepositoryInfo
}

