package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Engine is the workflow execution engine
type Engine struct {
	workflows map[string]*Workflow
	mutex     sync.RWMutex
}

// NewEngine creates a new workflow engine
func NewEngine() *Engine {
	return &Engine{
		workflows: make(map[string]*Workflow),
	}
}

// RegisterWorkflow registers a workflow with the engine
func (e *Engine) RegisterWorkflow(workflow *Workflow) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	// Check if workflow already exists
	if _, exists := e.workflows[workflow.ID]; exists {
		return errors.New("workflow already exists")
	}
	
	// Store the workflow
	e.workflows[workflow.ID] = workflow
	
	return nil
}

// GetWorkflow retrieves a workflow by ID
func (e *Engine) GetWorkflow(id string) (*Workflow, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// Check if workflow exists
	workflow, exists := e.workflows[id]
	if !exists {
		return nil, errors.New("workflow not found")
	}
	
	return workflow, nil
}

// ExecuteWorkflow executes a workflow with the given input
func (e *Engine) ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (map[string]interface{}, error) {
	// Get the workflow
	workflow, err := e.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}
	
	// Create a workflow context to track execution state
	workflowCtx := NewWorkflowContext(workflow, input)
	
	// Execute the workflow
	return workflowCtx.Execute(ctx)
}

// Workflow represents a workflow definition
type Workflow struct {
	ID          string
	Name        string
	Description string
	Type        string  // "prompt_chaining", "routing", "parallelization", "orchestrator_workers", "evaluator_optimizer"
	Steps       []Step
	
	// Type-specific configurations
	RouterStepIndex  int                    // For "routing" workflows
	ParallelGroups   map[string][]int       // For "parallelization" workflows
	OrchestratorStep int                    // For "orchestrator_workers" workflows
	EvaluatorStep    int                    // For "evaluator_optimizer" workflows
}

// Step represents a workflow step
type Step struct {
	ID          string
	Name        string
	Type        string  // "agent", "llm", "tool", "conditional", "subworkflow"
	AgentID     string  // For "agent" steps
	PromptTemplate string  // For "llm" steps
	ToolName    string  // For "tool" steps
	Condition   string  // For "conditional" steps
	WorkflowID  string  // For "subworkflow" steps
	NextSteps   []string  // IDs of next steps
}

// WorkflowContext holds the execution context for a workflow
type WorkflowContext struct {
	Workflow    *Workflow
	Input       map[string]interface{}
	Output      map[string]interface{}
	StepOutputs map[string]interface{}
	CurrentStep string
	Visited     map[string]bool
}

// NewWorkflowContext creates a new workflow context
func NewWorkflowContext(workflow *Workflow, input map[string]interface{}) *WorkflowContext {
	return &WorkflowContext{
		Workflow:    workflow,
		Input:       input,
		Output:      make(map[string]interface{}),
		StepOutputs: make(map[string]interface{}),
		Visited:     make(map[string]bool),
	}
}

// Execute executes the workflow
func (wc *WorkflowContext) Execute(ctx context.Context) (map[string]interface{}, error) {
	// Execute the workflow based on its type
	switch wc.Workflow.Type {
	case "prompt_chaining":
		return wc.executePromptChaining(ctx)
	case "routing":
		return wc.executeRouting(ctx)
	case "parallelization":
		return wc.executeParallelization(ctx)
	case "orchestrator_workers":
		return wc.executeOrchestratorWorkers(ctx)
	case "evaluator_optimizer":
		return wc.executeEvaluatorOptimizer(ctx)
	default:
		return nil, fmt.Errorf("unsupported workflow type: %s", wc.Workflow.Type)
	}
}

// executePromptChaining executes a prompt chaining workflow
func (wc *WorkflowContext) executePromptChaining(ctx context.Context) (map[string]interface{}, error) {
	// Start with the first step
	currentStep := wc.Workflow.Steps[0]
	wc.CurrentStep = currentStep.ID
	
	// Execute steps sequentially
	for {
		// Mark the step as visited
		wc.Visited[currentStep.ID] = true
		
		// Execute the step
		stepOutput, err := wc.executeStep(ctx, currentStep)
		if err != nil {
			return nil, err
		}
		
		// Store the step output
		wc.StepOutputs[currentStep.ID] = stepOutput
		
		// Add to the final output
		for k, v := range stepOutput {
			wc.Output[k] = v
		}
		
		// Check if we have next steps
		if len(currentStep.NextSteps) == 0 {
			// This is the last step
			break
		}
		
		// Get the next step
		nextStepID := currentStep.NextSteps[0]
		var found bool
		for _, step := range wc.Workflow.Steps {
			if step.ID == nextStepID {
				currentStep = step
				wc.CurrentStep = step.ID
				found = true
				break
			}
		}
		
		if !found {
			return nil, fmt.Errorf("step not found: %s", nextStepID)
		}
	}
	
	return wc.Output, nil
}

// executeRouting executes a routing workflow
func (wc *WorkflowContext) executeRouting(ctx context.Context) (map[string]interface{}, error) {
	// Get the router step
	routerStep := wc.Workflow.Steps[wc.Workflow.RouterStepIndex]
	wc.CurrentStep = routerStep.ID
	
	// Execute the router step
	routerOutput, err := wc.executeStep(ctx, routerStep)
	if err != nil {
		return nil, err
	}
	
	// Store the router output
	wc.StepOutputs[routerStep.ID] = routerOutput
	
	// Get the route from the output
	route, ok := routerOutput["route"].(string)
	if !ok {
		return nil, errors.New("router step did not return a route")
	}
	
	// Find the next step based on the route
	var nextStep *Step
	for _, step := range wc.Workflow.Steps {
		if step.ID == route {
			nextStep = &step
			break
		}
	}
	
	if nextStep == nil {
		return nil, fmt.Errorf("route not found: %s", route)
	}
	
	// Execute the next step
	wc.CurrentStep = nextStep.ID
	stepOutput, err := wc.executeStep(ctx, *nextStep)
	if err != nil {
		return nil, err
	}
	
	// Store the step output
	wc.StepOutputs[nextStep.ID] = stepOutput
	
	// Add to the final output
	for k, v := range stepOutput {
		wc.Output[k] = v
	}
	
	return wc.Output, nil
}

// executeParallelization executes a parallelization workflow
func (wc *WorkflowContext) executeParallelization(ctx context.Context) (map[string]interface{}, error) {
	// Execute each parallel group
	for groupName, stepIndices := range wc.Workflow.ParallelGroups {
		// Create a wait group for parallel execution
		var wg sync.WaitGroup
		results := make(map[string]interface{})
		errors := make(map[string]error)
		var mutex sync.Mutex
		
		// Add tasks to the wait group
		for _, idx := range stepIndices {
			step := wc.Workflow.Steps[idx]
			wg.Add(1)
			
			// Execute the step in a goroutine
			go func(s Step) {
				defer wg.Done()
				
				// Execute the step
				stepOutput, err := wc.executeStep(ctx, s)
				
				// Store the result
				mutex.Lock()
				if err != nil {
					errors[s.ID] = err
				} else {
					results[s.ID] = stepOutput
					wc.StepOutputs[s.ID] = stepOutput
				}
				mutex.Unlock()
			}(step)
		}
		
		// Wait for all tasks to complete
		wg.Wait()
		
		// Check if any tasks failed
		if len(errors) > 0 {
			// Return the first error
			for _, err := range errors {
				return nil, err
			}
		}
		
		// Combine the results
		groupResults := make(map[string]interface{})
		for stepID, result := range results {
			resultMap, ok := result.(map[string]interface{})
			if ok {
				for k, v := range resultMap {
					groupResults[k] = v
				}
			} else {
				groupResults[stepID] = result
			}
		}
		
		// Store the group results
		wc.Output[groupName] = groupResults
	}
	
	return wc.Output, nil
}

// executeOrchestratorWorkers executes an orchestrator-workers workflow
func (wc *WorkflowContext) executeOrchestratorWorkers(ctx context.Context) (map[string]interface{}, error) {
	// Get the orchestrator step
	orchestratorStep := wc.Workflow.Steps[wc.Workflow.OrchestratorStep]
	wc.CurrentStep = orchestratorStep.ID
	
	// Execute the orchestrator step to get the plan
	planOutput, err := wc.executeStep(ctx, orchestratorStep)
	if err != nil {
		return nil, err
	}
	
	// Store the orchestrator output
	wc.StepOutputs[orchestratorStep.ID] = planOutput
	
	// Extract the worker assignments from the plan
	workerAssignments, ok := planOutput["worker_assignments"].(map[string]interface{})
	if !ok {
		return nil, errors.New("orchestrator did not return worker assignments")
	}
	
	// Execute each worker task
	workerResults := make(map[string]interface{})
	for workerID, assignment := range workerAssignments {
		// Find the worker step
		var workerStep *Step
		for _, step := range wc.Workflow.Steps {
			if step.ID == workerID {
				workerStep = &step
				break
			}
		}
		
		if workerStep == nil {
			return nil, fmt.Errorf("worker not found: %s", workerID)
		}
		
		// Create a context with the assignment
		workerCtx := context.WithValue(ctx, "assignment", assignment)
		
		// Execute the worker step
		wc.CurrentStep = workerStep.ID
		workerOutput, err := wc.executeStep(workerCtx, *workerStep)
		if err != nil {
			return nil, err
		}
		
		// Store the worker output
		wc.StepOutputs[workerStep.ID] = workerOutput
		workerResults[workerID] = workerOutput
	}
	
	// Execute the orchestrator again to synthesize results
	synthesisCtx := context.WithValue(ctx, "worker_results", workerResults)
	wc.CurrentStep = orchestratorStep.ID
	synthesisOutput, err := wc.executeStep(synthesisCtx, orchestratorStep)
	if err != nil {
		return nil, err
	}
	
	// Store the synthesis output
	wc.Output = synthesisOutput
	
	return wc.Output, nil
}

// executeEvaluatorOptimizer executes an evaluator-optimizer workflow
func (wc *WorkflowContext) executeEvaluatorOptimizer(ctx context.Context) (map[string]interface{}, error) {
	// Get the evaluator step
	evaluatorStep := wc.Workflow.Steps[wc.Workflow.EvaluatorStep]
	
	// Find the optimizer step (first step that's not the evaluator)
	var optimizerStep *Step
	for _, step := range wc.Workflow.Steps {
		if step.ID != evaluatorStep.ID {
			optimizerStep = &step
			break
		}
	}
	
	if optimizerStep == nil {
		return nil, errors.New("optimizer step not found")
	}
	
	// Initialize with the optimizer
	wc.CurrentStep = optimizerStep.ID
	optimizerOutput, err := wc.executeStep(ctx, *optimizerStep)
	if err != nil {
		return nil, err
	}
	
	// Store the optimizer output
	wc.StepOutputs[optimizerStep.ID] = optimizerOutput
	
	// Maximum number of iterations
	maxIterations := 5
	
	// Iteratively evaluate and optimize
	for i := 0; i < maxIterations; i++ {
		// Evaluate the current output
		wc.CurrentStep = evaluatorStep.ID
		evaluationCtx := context.WithValue(ctx, "current_solution", optimizerOutput)
		evaluationOutput, err := wc.executeStep(evaluationCtx, evaluatorStep)
		if err != nil {
			return nil, err
		}
		
		// Store the evaluation output
		wc.StepOutputs[evaluatorStep.ID] = evaluationOutput
		
		// Check if we've reached an acceptable solution
		acceptable, ok := evaluationOutput["acceptable"].(bool)
		if ok && acceptable {
			// Solution is acceptable, return it
			wc.Output = optimizerOutput
			return wc.Output, nil
		}
		
		// Optimize again with the evaluation feedback
		wc.CurrentStep = optimizerStep.ID
		optimizationCtx := context.WithValue(ctx, "feedback", evaluationOutput)
		optimizerOutput, err = wc.executeStep(optimizationCtx, *optimizerStep)
		if err != nil {
			return nil, err
		}
		
		// Store the optimizer output
		wc.StepOutputs[optimizerStep.ID] = optimizerOutput
	}
	
	// Return the last optimizer output
	wc.Output = optimizerOutput
	
	return wc.Output, nil
}

// executeStep executes a single workflow step
func (wc *WorkflowContext) executeStep(ctx context.Context, step Step) (map[string]interface{}, error) {
	// First, check if we have a step executor in the context
	// This allows for dependency injection of executors
	if executor, ok := ctx.Value("step_executor").(func(context.Context, Step, map[string]interface{}) (map[string]interface{}, error)); ok {
		return executor(ctx, step, wc.Input)
	}
	
	// Execute the step based on its type
	switch step.Type {
	case "agent":
		// Check if we have an agent executor in the context
		agentExecutor, ok := ctx.Value("agent_executor").(func(context.Context, string, map[string]interface{}) (map[string]interface{}, error))
		if ok {
			// We have an agent executor, use it
			return agentExecutor(ctx, step.AgentID, wc.Input)
		}
		
		// Check if we have an agent manager in the context
		agentManager, ok := ctx.Value("agent_manager").(interface{
			GetAgent(ctx context.Context, agentID string) (interface{
				ExecuteTask(ctx context.Context, task interface{}) (map[string]interface{}, error)
			}, error)
		})
		
		if ok {
			// Get the agent from the manager
			agent, err := agentManager.GetAgent(ctx, step.AgentID)
			if err != nil {
				return nil, fmt.Errorf("failed to get agent %s: %w", step.AgentID, err)
			}
			
			// Create a task from the input
			task := map[string]interface{}{
				"Description": fmt.Sprintf("Execute workflow step: %s", step.Name),
				"Input":       wc.Input,
				"Metadata": map[string]interface{}{
					"workflow_id":  wc.Workflow.ID,
					"workflow_name": wc.Workflow.Name,
					"step_id":      step.ID,
					"step_name":    step.Name,
				},
			}
			
			// Execute the task on the agent
			return agent.ExecuteTask(ctx, task)
		}
		
		// For this implementation, since we don't have access to the agent manager,
		// we'll just return a placeholder result
		return map[string]interface{}{
			"status": "pending",
			"message": fmt.Sprintf("Agent %s task waiting to be processed", step.AgentID),
			"agent_id": step.AgentID,
			"input": wc.Input,
		}, nil
		
	case "llm":
		// Check if we have an LLM provider in the context
		llmProvider, ok := ctx.Value("llm_provider").(interface{
			Complete(ctx context.Context, prompt string, params map[string]interface{}) (interface{}, error)
		})
		
		if ok {
			// Process the prompt template with the input data
			prompt := processTemplate(step.PromptTemplate, wc.Input)
			
			// Call the LLM
			response, err := llmProvider.Complete(ctx, prompt, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to execute LLM step: %w", err)
			}
			
			// Return the response
			return map[string]interface{}{
				"result": response,
				"status": "completed",
			}, nil
		}
		
		// In a real implementation, this would call the LLM
		return map[string]interface{}{
			"result": fmt.Sprintf("LLM execution result for prompt: %s", step.PromptTemplate),
			"status": "completed",
		}, nil
		
	case "tool":
		// Check if we have a tool registry in the context
		toolRegistry, ok := ctx.Value("tool_registry").(interface{
			GetTool(name string) (interface{
				Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
			}, error)
		})
		
		if ok {
			// Get the tool from the registry
			tool, err := toolRegistry.GetTool(step.ToolName)
			if err != nil {
				return nil, fmt.Errorf("failed to get tool %s: %w", step.ToolName, err)
			}
			
			// Execute the tool with the input as arguments
			result, err := tool.Execute(ctx, wc.Input)
			if err != nil {
				return nil, fmt.Errorf("failed to execute tool %s: %w", step.ToolName, err)
			}
			
			// Return the result
			return map[string]interface{}{
				"result": result,
				"status": "completed",
			}, nil
		}
		
		// In a real implementation, this would call the tool
		return map[string]interface{}{
			"result": fmt.Sprintf("Tool %s execution result", step.ToolName),
			"status": "completed",
		}, nil
		
	case "conditional":
		// In a real implementation, this would evaluate the condition
		// and return different results based on the condition
		condition := step.Condition
		result := evaluateCondition(condition, wc.Input)
		
		return map[string]interface{}{
			"result": result,
			"condition": condition,
			"evaluated": true,
			"status": "completed",
		}, nil
		
	case "subworkflow":
		// Check if we have a workflow engine in the context
		workflowEngine, ok := ctx.Value("workflow_engine").(interface{
			ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (map[string]interface{}, error)
		})
		
		if ok {
			// Execute the subworkflow
			result, err := workflowEngine.ExecuteWorkflow(ctx, step.WorkflowID, wc.Input)
			if err != nil {
				return nil, fmt.Errorf("failed to execute subworkflow %s: %w", step.WorkflowID, err)
			}
			
			// Return the result
			return map[string]interface{}{
				"result": result,
				"status": "completed",
			}, nil
		}
		
		// In a real implementation, this would execute the subworkflow
		return map[string]interface{}{
			"result": fmt.Sprintf("Subworkflow %s execution result", step.WorkflowID),
			"status": "completed",
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// processTemplate processes a prompt template with input data
// replacing {{variable}} with values from the input
func processTemplate(template string, input map[string]interface{}) string {
	result := template
	
	// Replace variables in the template
	for key, value := range input {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.Replace(result, placeholder, valueStr, -1)
	}
	
	return result
}

// evaluateCondition evaluates a condition expression against input data
// This is a simple implementation - in a real system you'd want a more
// sophisticated expression evaluator
func evaluateCondition(condition string, input map[string]interface{}) bool {
	// For now, just check if any part of the condition is in the input
	// This is a very basic implementation
	parts := strings.Split(condition, " ")
	
	for _, part := range parts {
		// Remove any operators or special characters
		cleanPart := strings.Trim(part, "=!<>()&|")
		
		// Check if this part is a key in the input
		if _, ok := input[cleanPart]; ok {
			return true
		}
	}
	
	return false
}
