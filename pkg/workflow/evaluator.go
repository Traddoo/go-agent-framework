package workflow

import (
	"github.com/google/uuid"
)

// NewEvaluatorWorkflow creates a new evaluator-optimizer workflow
func NewEvaluatorWorkflow(name string, description string, evaluatorStep Step, optimizerStep Step) *Workflow {
	// Create steps array with evaluator and optimizer
	steps := []Step{evaluatorStep, optimizerStep}
	
	// Create a new workflow
	workflow := &Workflow{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		Type:           "evaluator_optimizer",
		Steps:          steps,
		EvaluatorStep:  0, // Evaluator is always the first step
	}
	
	return workflow
}

// EvaluatorBuilder helps build evaluator-optimizer workflows
type EvaluatorBuilder struct {
	name          string
	description   string
	evaluatorStep Step
	optimizerStep Step
}

// NewEvaluatorBuilder creates a new evaluator builder
func NewEvaluatorBuilder(name string, description string) *EvaluatorBuilder {
	return &EvaluatorBuilder{
		name:        name,
		description: description,
	}
}

// SetEvaluatorAgent sets an agent as the evaluator
func (b *EvaluatorBuilder) SetEvaluatorAgent(name string, agentID string) *EvaluatorBuilder {
	b.evaluatorStep = Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	return b
}

// SetEvaluatorLLM sets an LLM as the evaluator
func (b *EvaluatorBuilder) SetEvaluatorLLM(name string, promptTemplate string) *EvaluatorBuilder {
	b.evaluatorStep = Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	return b
}

// SetEvaluatorTool sets a tool as the evaluator
func (b *EvaluatorBuilder) SetEvaluatorTool(name string, toolName string) *EvaluatorBuilder {
	b.evaluatorStep = Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	}
	return b
}

// SetOptimizerAgent sets an agent as the optimizer
func (b *EvaluatorBuilder) SetOptimizerAgent(name string, agentID string) *EvaluatorBuilder {
	b.optimizerStep = Step{
		ID:      uuid.New().String(),
		Name:    name,
		Type:    "agent",
		AgentID: agentID,
	}
	return b
}

// SetOptimizerLLM sets an LLM as the optimizer
func (b *EvaluatorBuilder) SetOptimizerLLM(name string, promptTemplate string) *EvaluatorBuilder {
	b.optimizerStep = Step{
		ID:             uuid.New().String(),
		Name:           name,
		Type:           "llm",
		PromptTemplate: promptTemplate,
	}
	return b
}

// SetOptimizerTool sets a tool as the optimizer
func (b *EvaluatorBuilder) SetOptimizerTool(name string, toolName string) *EvaluatorBuilder {
	b.optimizerStep = Step{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     "tool",
		ToolName: toolName,
	}
	return b
}

// Build builds the workflow
func (b *EvaluatorBuilder) Build() *Workflow {
	return NewEvaluatorWorkflow(b.name, b.description, b.evaluatorStep, b.optimizerStep)
}
