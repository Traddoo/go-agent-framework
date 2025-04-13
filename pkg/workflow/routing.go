package workflow

import (
	"github.com/google/uuid"
)

// NewRoutingWorkflow creates a new routing workflow
func NewRoutingWorkflow(name string, description string, routerStep Step, routeSteps []Step) *Workflow {
	// Create steps array with router as first step
	steps := make([]Step, 1+len(routeSteps))
	steps[0] = routerStep
	copy(steps[1:], routeSteps)
	
	// Create a new workflow
	workflow := &Workflow{
		ID:              uuid.New().String(),
		Name:            name,
		Description:     description,
		Type:            "routing",
		Steps:           steps,
		RouterStepIndex: 0, // Router is always the first step
	}
	
	return workflow
}

// RoutingBuilder helps build routing workflows
type RoutingBuilder struct {
	name        string
	description string
	routerStep  Step
	routeSteps  []Step
}

// NewRoutingBuilder creates a new routing builder
func NewRoutingBuilder(name string, description string) *RoutingBuilder {
	return &RoutingBuilder{
		name:        name,
		description: description,
		routeSteps:  make([]Step, 0),
	}
}

// SetRouterAgent sets an agent as the router
func (b *RoutingBuilder) SetRouterAgent(name string, agentID string) *RoutingBuilder {
	b.routerStep = Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	return b
}

// SetRouterLLM sets an LLM as the router
func (b *RoutingBuilder) SetRouterLLM(name string, promptTemplate string) *RoutingBuilder {
	b.routerStep = Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	return b
}

// AddAgentRoute adds an agent as a possible route
func (b *RoutingBuilder) AddAgentRoute(name string, agentID string) *RoutingBuilder {
	step := Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	b.routeSteps = append(b.routeSteps, step)
	return b
}

// AddLLMRoute adds an LLM as a possible route
func (b *RoutingBuilder) AddLLMRoute(name string, promptTemplate string) *RoutingBuilder {
	step := Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	b.routeSteps = append(b.routeSteps, step)
	return b
}

// AddToolRoute adds a tool as a possible route
func (b *RoutingBuilder) AddToolRoute(name string, toolName string) *RoutingBuilder {
	step := Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	}
	b.routeSteps = append(b.routeSteps, step)
	return b
}

// AddSubworkflowRoute adds a subworkflow as a possible route
func (b *RoutingBuilder) AddSubworkflowRoute(name string, workflowID string) *RoutingBuilder {
	step := Step{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       "subworkflow",
		WorkflowID: workflowID,
	}
	b.routeSteps = append(b.routeSteps, step)
	return b
}

// Build builds the workflow
func (b *RoutingBuilder) Build() *Workflow {
	return NewRoutingWorkflow(b.name, b.description, b.routerStep, b.routeSteps)
}
