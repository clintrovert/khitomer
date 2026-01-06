package planner

import (
	"github.com/clintrovert/khitomer/pkg/types"
)

// Planner interface for generating implementation plans
type Planner interface {
	Plan(task *types.Task) (*types.ImplementationPlan, error)
}

