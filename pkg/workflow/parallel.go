package workflow

import (
	"github.com/google/uuid"
)

// NewParallelizationWorkflow creates a new parallelization workflow
func NewParallelizationWorkflow(name string, description string, parallelGroups map[string][]Step) *Workflow {
	// Flatten all steps for storage
	var steps []Step
	groupIndices := make(map[string][]int)
	
	for groupName, groupSteps := range parallelGroups {
		indices := make([]int, len(groupSteps))
		startIdx := len(steps)
		
		for i, step := range groupSteps {
			steps = append(steps, step)
			indices[i] = startIdx + i
		}
		
		groupIndices[groupName] = indices
	}
	
	// Create a new workflow
	workflow := &Workflow{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		Type:           "parallelization",
		Steps:          steps,
		ParallelGroups: groupIndices,
	}
	
	return workflow
}

// ParallelizationBuilder helps build parallelization workflows
type ParallelizationBuilder struct {
	name          string
	description   string
	parallelGroups map[string][]Step
}

// NewParallelizationBuilder creates a new parallelization builder
func NewParallelizationBuilder(name string, description string) *ParallelizationBuilder {
	return &ParallelizationBuilder{
		name:          name,
		description:   description,
		parallelGroups: make(map[string][]Step),
	}
}

// CreateGroup creates a new parallel group
func (b *ParallelizationBuilder) CreateGroup(groupName string) *ParallelizationBuilder {
	if _, exists := b.parallelGroups[groupName]; !exists {
		b.parallelGroups[groupName] = make([]Step, 0)
	}
	return b
}

// AddAgentStep adds an agent step to a group
func (b *ParallelizationBuilder) AddAgentStep(groupName string, name string, agentID string) *ParallelizationBuilder {
	if _, exists := b.parallelGroups[groupName]; !exists {
		b.CreateGroup(groupName)
	}
	
	step := Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	
	b.parallelGroups[groupName] = append(b.parallelGroups[groupName], step)
	return b
}

// AddLLMStep adds an LLM step to a group
func (b *ParallelizationBuilder) AddLLMStep(groupName string, name string, promptTemplate string) *ParallelizationBuilder {
	if _, exists := b.parallelGroups[groupName]; !exists {
		b.CreateGroup(groupName)
	}
	
	step := Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	
	b.parallelGroups[groupName] = append(b.parallelGroups[groupName], step)
	return b
}

// AddToolStep adds a tool step to a group
func (b *ParallelizationBuilder) AddToolStep(groupName string, name string, toolName string) *ParallelizationBuilder {
	if _, exists := b.parallelGroups[groupName]; !exists {
		b.CreateGroup(groupName)
	}
	
	step := Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	}
	
	b.parallelGroups[groupName] = append(b.parallelGroups[groupName], step)
	return b
}

// AddSubworkflowStep adds a subworkflow step to a group
func (b *ParallelizationBuilder) AddSubworkflowStep(groupName string, name string, workflowID string) *ParallelizationBuilder {
	if _, exists := b.parallelGroups[groupName]; !exists {
		b.CreateGroup(groupName)
	}
	
	step := Step{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       "subworkflow",
		WorkflowID: workflowID,
	}
	
	b.parallelGroups[groupName] = append(b.parallelGroups[groupName], step)
	return b
}

// Build builds the workflow
func (b *ParallelizationBuilder) Build() *Workflow {
	return NewParallelizationWorkflow(b.name, b.description, b.parallelGroups)
}
