package workflow

import (
	"github.com/google/uuid"
)

// NewOrchestratorWorkflow creates a new orchestrator-workers workflow
func NewOrchestratorWorkflow(name string, description string, orchestratorStep Step, workerSteps []Step) *Workflow {
	// Create steps array with orchestrator and worker steps
	steps := make([]Step, 1+len(workerSteps))
	steps[0] = orchestratorStep
	copy(steps[1:], workerSteps)
	
	// Create a new workflow
	workflow := &Workflow{
		ID:              uuid.New().String(),
		Name:            name,
		Description:     description,
		Type:            "orchestrator_workers",
		Steps:           steps,
		OrchestratorStep: 0, // Orchestrator is always the first step
	}
	
	return workflow
}

// OrchestratorBuilder helps build orchestrator-workers workflows
type OrchestratorBuilder struct {
	name             string
	description      string
	orchestratorStep Step
	workerSteps      []Step
}

// NewOrchestratorBuilder creates a new orchestrator builder
func NewOrchestratorBuilder(name string, description string) *OrchestratorBuilder {
	return &OrchestratorBuilder{
		name:        name,
		description: description,
		workerSteps: make([]Step, 0),
	}
}

// SetOrchestratorAgent sets an agent as the orchestrator
func (b *OrchestratorBuilder) SetOrchestratorAgent(name string, agentID string) *OrchestratorBuilder {
	b.orchestratorStep = Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	return b
}

// SetOrchestratorLLM sets an LLM as the orchestrator
func (b *OrchestratorBuilder) SetOrchestratorLLM(name string, promptTemplate string) *OrchestratorBuilder {
	b.orchestratorStep = Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	return b
}

// AddWorkerAgent adds an agent as a worker
func (b *OrchestratorBuilder) AddWorkerAgent(name string, agentID string) *OrchestratorBuilder {
	step := Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	b.workerSteps = append(b.workerSteps, step)
	return b
}

// AddWorkerLLM adds an LLM as a worker
func (b *OrchestratorBuilder) AddWorkerLLM(name string, promptTemplate string) *OrchestratorBuilder {
	step := Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	b.workerSteps = append(b.workerSteps, step)
	return b
}

// AddWorkerTool adds a tool as a worker
func (b *OrchestratorBuilder) AddWorkerTool(name string, toolName string) *OrchestratorBuilder {
	step := Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	}
	b.workerSteps = append(b.workerSteps, step)
	return b
}

// AddWorkerSubworkflow adds a subworkflow as a worker
func (b *OrchestratorBuilder) AddWorkerSubworkflow(name string, workflowID string) *OrchestratorBuilder {
	step := Step{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       "subworkflow",
		WorkflowID: workflowID,
	}
	b.workerSteps = append(b.workerSteps, step)
	return b
}

// Build builds the workflow
func (b *OrchestratorBuilder) Build() *Workflow {
	return NewOrchestratorWorkflow(b.name, b.description, b.orchestratorStep, b.workerSteps)
}
