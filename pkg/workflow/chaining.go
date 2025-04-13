package workflow

import (
	"github.com/google/uuid"
)

// NewPromptChainingWorkflow creates a new prompt chaining workflow
func NewPromptChainingWorkflow(name string, description string, steps []Step) *Workflow {
	// Create a new workflow
	workflow := &Workflow{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Type:        "prompt_chaining",
		Steps:       steps,
	}
	
	// Set up the step chain
	for i := 0; i < len(steps)-1; i++ {
		// Each step points to the next one
		steps[i].NextSteps = []string{steps[i+1].ID}
	}
	
	return workflow
}

// PromptChainingBuilder helps build prompt chaining workflows
type PromptChainingBuilder struct {
	name        string
	description string
	steps       []Step
}

// NewPromptChainingBuilder creates a new prompt chaining builder
func NewPromptChainingBuilder(name string, description string) *PromptChainingBuilder {
	return &PromptChainingBuilder{
		name:        name,
		description: description,
		steps:       make([]Step, 0),
	}
}

// AddAgentStep adds an agent step to the workflow
func (b *PromptChainingBuilder) AddAgentStep(name string, agentID string) *PromptChainingBuilder {
	b.steps = append(b.steps, Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	})
	return b
}

// AddLLMStep adds an LLM step to the workflow
func (b *PromptChainingBuilder) AddLLMStep(name string, promptTemplate string) *PromptChainingBuilder {
	b.steps = append(b.steps, Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	})
	return b
}

// AddToolStep adds a tool step to the workflow
func (b *PromptChainingBuilder) AddToolStep(name string, toolName string) *PromptChainingBuilder {
	b.steps = append(b.steps, Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	})
	return b
}

// AddConditionalStep adds a conditional step to the workflow
func (b *PromptChainingBuilder) AddConditionalStep(name string, condition string) *PromptChainingBuilder {
	b.steps = append(b.steps, Step{
		ID:        uuid.New().String(),
		Name:      name,
		Type:      "conditional",
		Condition: condition,
	})
	return b
}

// AddSubworkflowStep adds a subworkflow step to the workflow
func (b *PromptChainingBuilder) AddSubworkflowStep(name string, workflowID string) *PromptChainingBuilder {
	b.steps = append(b.steps, Step{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       "subworkflow",
		WorkflowID: workflowID,
	})
	return b
}

// Build builds the workflow
func (b *PromptChainingBuilder) Build() *Workflow {
	return NewPromptChainingWorkflow(b.name, b.description, b.steps)
}
