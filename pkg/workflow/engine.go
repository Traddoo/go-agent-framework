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
    
    if _, exists := e.workflows[workflow.ID]; exists {
        return errors.New("workflow already exists")
    }
    
    e.workflows[workflow.ID] = workflow
    return nil
}

// GetWorkflow retrieves a workflow by ID
func (e *Engine) GetWorkflow(id string) (*Workflow, error) {
    e.mutex.RLock()
    defer e.mutex.RUnlock()
    
    workflow, exists := e.workflows[id]
    if !exists {
        return nil, errors.New("workflow not found")
    }
    
    return workflow, nil
}

// ExecuteWorkflow executes a workflow with the given input
func (e *Engine) ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (*Result, error) {
    workflow, err := e.GetWorkflow(workflowID)
    if err != nil {
        return nil, err
    }
    
    result := &Result{
        WorkflowID: workflowID,
        status: Status{
            State:    "running",
            Progress: 0.0,
        },
    }
    
    workflowCtx := NewWorkflowContext(workflow, input)
    
    output, err := workflowCtx.Execute(ctx)
    if err != nil {
        result.SetStatus(Status{
            State:    "failed",
            Progress: workflowCtx.calculateProgress(),
            Error:    err,
        })
        return result, err
    }
    
    result.SetStatus(Status{
        State:    "completed",
        Progress: 1.0,
    })
    
    // Store the output in the context
    workflowCtx.Output = output
    
    return result, nil
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
        // Default to sequential execution
        return wc.executeSequential(ctx)
    }
}

// executeSequential executes steps sequentially
func (wc *WorkflowContext) executeSequential(ctx context.Context) (map[string]interface{}, error) {
    for _, step := range wc.Workflow.Steps {
        wc.CurrentStep = step.ID
        wc.Visited[step.ID] = true
        
        // Execute all tasks in the step
        if step.Parallel {
            taskOutput, err := wc.executeParallelTasks(ctx, step)
            if err != nil {
                return nil, fmt.Errorf("failed to execute parallel tasks in step %s: %w", step.ID, err)
            }
            wc.StepOutputs[step.ID] = taskOutput
            // Merge task output into workflow output
            for k, v := range taskOutput {
                wc.Output[k] = v
            }
        } else {
            taskOutput, err := wc.executeStep(ctx, step)
            if err != nil {
                return nil, fmt.Errorf("failed to execute step %s: %w", step.ID, err)
            }
            wc.StepOutputs[step.ID] = taskOutput
            // Merge task output into workflow output
            for k, v := range taskOutput {
                wc.Output[k] = v
            }
        }
    }
    
    return wc.Output, nil
}

// executeParallelTasks executes tasks in parallel
func (wc *WorkflowContext) executeParallelTasks(ctx context.Context, step Step) (map[string]interface{}, error) {
    var wg sync.WaitGroup
    results := make(map[string]interface{})
    errors := make(map[string]error)
    var mutex sync.Mutex
    
    for _, task := range step.Tasks {
        wg.Add(1)
        go func(t Task) {
            defer wg.Done()
            
            taskInput := wc.prepareTaskInput(t.Input)
            output, err := wc.executeTask(ctx, t.AgentID, taskInput)
            
            mutex.Lock()
            if err != nil {
                errors[t.AgentID] = err
            } else {
                results[t.AgentID] = output
            }
            mutex.Unlock()
        }(task)
    }
    
    wg.Wait()
    
    if len(errors) > 0 {
        // Return the first error encountered
        for _, err := range errors {
            return nil, err
        }
    }
    
    return results, nil
}

// executeStep executes a single workflow step
func (wc *WorkflowContext) executeStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    switch step.Type {
    case "agent":
        return wc.executeAgentStep(ctx, step)
    case "llm":
        return wc.executeLLMStep(ctx, step)
    case "tool":
        return wc.executeToolStep(ctx, step)
    case "conditional":
        return wc.executeConditionalStep(ctx, step)
    case "subworkflow":
        return wc.executeSubworkflowStep(ctx, step)
    default:
        return nil, fmt.Errorf("unsupported step type: %s", step.Type)
    }
}

// executeAgentStep executes an agent step
func (wc *WorkflowContext) executeAgentStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    agentExecutor, ok := ctx.Value("agent_executor").(func(context.Context, string, map[string]interface{}) (map[string]interface{}, error))
    if ok {
        return agentExecutor(ctx, step.AgentID, wc.Input)
    }
    
    return nil, errors.New("agent executor not found in context")
}

// executeLLMStep executes an LLM step
func (wc *WorkflowContext) executeLLMStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    llmProvider, ok := ctx.Value("llm_provider").(interface {
        Complete(ctx context.Context, prompt string, params map[string]interface{}) (interface{}, error)
    })
    if !ok {
        return nil, errors.New("llm provider not found in context")
    }
    
    prompt := processTemplate(step.PromptTemplate, wc.Input)
    response, err := llmProvider.Complete(ctx, prompt, nil)
    if err != nil {
        return nil, fmt.Errorf("llm execution failed: %w", err)
    }
    
    return map[string]interface{}{
        "result": response,
        "status": "completed",
    }, nil
}

// executeToolStep executes a tool step
func (wc *WorkflowContext) executeToolStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    toolRegistry, ok := ctx.Value("tool_registry").(interface {
        GetTool(name string) (interface {
            Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
        }, error)
    })
    if !ok {
        return nil, errors.New("tool registry not found in context")
    }
    
    tool, err := toolRegistry.GetTool(step.ToolName)
    if err != nil {
        return nil, fmt.Errorf("failed to get tool: %w", err)
    }
    
    result, err := tool.Execute(ctx, wc.Input)
    if err != nil {
        return nil, fmt.Errorf("tool execution failed: %w", err)
    }
    
    return map[string]interface{}{
        "result": result,
        "status": "completed",
    }, nil
}

// executeConditionalStep executes a conditional step
func (wc *WorkflowContext) executeConditionalStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    if evaluateCondition(step.Condition, wc.Input) {
        return wc.executeStep(ctx, step)
    }
    return map[string]interface{}{"status": "skipped"}, nil
}

// executeSubworkflowStep executes a subworkflow step
func (wc *WorkflowContext) executeSubworkflowStep(ctx context.Context, step Step) (map[string]interface{}, error) {
    workflowEngine, ok := ctx.Value("workflow_engine").(*Engine)
    if !ok {
        return nil, errors.New("workflow engine not found in context")
    }
    
    result, err := workflowEngine.ExecuteWorkflow(ctx, step.WorkflowID, wc.Input)
    if err != nil {
        return nil, fmt.Errorf("subworkflow execution failed: %w", err)
    }
    
    return map[string]interface{}{
        "result": result,
        "status": "completed",
    }, nil
}

// prepareTaskInput prepares the input for a task by merging with workflow input
func (wc *WorkflowContext) prepareTaskInput(taskInput map[string]interface{}) map[string]interface{} {
    input := make(map[string]interface{})
    
    // Copy workflow input
    for k, v := range wc.Input {
        input[k] = v
    }
    
    // Merge task input
    for k, v := range taskInput {
        input[k] = v
    }
    
    return input
}

// executeTask executes a task using an agent
func (wc *WorkflowContext) executeTask(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
    agentExecutor, ok := ctx.Value("agent_executor").(func(context.Context, string, map[string]interface{}) (map[string]interface{}, error))
    if !ok {
        return nil, errors.New("agent executor not found in context")
    }
    
    return agentExecutor(ctx, agentID, input)
}

// calculateProgress calculates the current progress of the workflow
func (wc *WorkflowContext) calculateProgress() float64 {
    if len(wc.Workflow.Steps) == 0 {
        return 1.0
    }
    
    completed := 0
    for _, step := range wc.Workflow.Steps {
        if wc.Visited[step.ID] {
            completed++
        }
    }
    
    return float64(completed) / float64(len(wc.Workflow.Steps))
}

// processTemplate processes a prompt template with input data
func processTemplate(template string, input map[string]interface{}) string {
    result := template
    
    for key, value := range input {
        placeholder := fmt.Sprintf("{{%s}}", key)
        valueStr := fmt.Sprintf("%v", value)
        result = strings.Replace(result, placeholder, valueStr, -1)
    }
    
    return result
}

// evaluateCondition evaluates a condition expression against input data
func evaluateCondition(condition string, input map[string]interface{}) bool {
    parts := strings.Split(condition, " ")
    
    for _, part := range parts {
        cleanPart := strings.Trim(part, "=!<>()&|")
        if _, ok := input[cleanPart]; ok {
            return true
        }
    }
    
    return false
}

// executePromptChaining executes a prompt chaining workflow
func (wc *WorkflowContext) executePromptChaining(ctx context.Context) (map[string]interface{}, error) {
    // Execute steps sequentially, passing output of each step as input to the next
    for i, step := range wc.Workflow.Steps {
        wc.CurrentStep = step.ID
        wc.Visited[step.ID] = true
        
        output, err := wc.executeStep(ctx, step)
        if err != nil {
            return nil, fmt.Errorf("failed to execute step %s: %w", step.ID, err)
        }
        
        // Store step output
        wc.StepOutputs[step.ID] = output
        
        // If not the last step, prepare input for next step
        if i < len(wc.Workflow.Steps)-1 {
            wc.Input = wc.prepareTaskInput(map[string]interface{}{
                "previous_step_output": output,
            })
        }
    }
    
    return wc.Output, nil
}

// executeRouting executes a routing workflow
func (wc *WorkflowContext) executeRouting(ctx context.Context) (map[string]interface{}, error) {
    if len(wc.Workflow.Steps) == 0 {
        return nil, errors.New("routing workflow must have at least one step")
    }
    
    // First step should be the router
    routerStep := wc.Workflow.Steps[0]
    wc.CurrentStep = routerStep.ID
    wc.Visited[routerStep.ID] = true
    
    // Execute router step
    routingResult, err := wc.executeStep(ctx, routerStep)
    if err != nil {
        return nil, fmt.Errorf("routing step failed: %w", err)
    }
    
    // Get the route from the result
    route, ok := routingResult["route"].(string)
    if !ok {
        return nil, errors.New("router step must return a 'route' string")
    }
    
    // Find and execute the matching step
    for _, step := range wc.Workflow.Steps[1:] {
        if step.ID == route {
            wc.CurrentStep = step.ID
            wc.Visited[step.ID] = true
            return wc.executeStep(ctx, step)
        }
    }
    
    return nil, fmt.Errorf("no matching route found for: %s", route)
}

// executeParallelization executes steps in parallel
func (wc *WorkflowContext) executeParallelization(ctx context.Context) (map[string]interface{}, error) {
    var wg sync.WaitGroup
    results := make(map[string]interface{})
    errors := make(map[string]error)
    var mutex sync.Mutex
    
    for _, step := range wc.Workflow.Steps {
        wg.Add(1)
        go func(s Step) {
            defer wg.Done()
            
            mutex.Lock()
            wc.Visited[s.ID] = true
            mutex.Unlock()
            
            output, err := wc.executeStep(ctx, s)
            
            mutex.Lock()
            if err != nil {
                errors[s.ID] = err
            } else {
                results[s.ID] = output
            }
            mutex.Unlock()
        }(step)
    }
    
    wg.Wait()
    
    if len(errors) > 0 {
        // Return the first error encountered
        for _, err := range errors {
            return nil, err
        }
    }
    
    return results, nil
}

// executeOrchestratorWorkers executes an orchestrator-workers workflow
func (wc *WorkflowContext) executeOrchestratorWorkers(ctx context.Context) (map[string]interface{}, error) {
    if len(wc.Workflow.Steps) < 2 {
        return nil, errors.New("orchestrator-workers workflow must have at least two steps")
    }
    
    // First step is the orchestrator
    orchestratorStep := wc.Workflow.Steps[0]
    wc.CurrentStep = orchestratorStep.ID
    wc.Visited[orchestratorStep.ID] = true
    
    // Execute orchestrator to get work assignments
    assignments, err := wc.executeStep(ctx, orchestratorStep)
    if err != nil {
        return nil, fmt.Errorf("orchestrator step failed: %w", err)
    }
    
    // Execute worker steps based on assignments
    workerResults := make(map[string]interface{})
    for _, step := range wc.Workflow.Steps[1:] {
        if assignment, ok := assignments[step.ID]; ok {
            wc.CurrentStep = step.ID
            wc.Visited[step.ID] = true
            
            // Prepare worker input
            workerInput := wc.prepareTaskInput(map[string]interface{}{
                "assignment": assignment,
            })
            
            result, err := wc.executeTask(ctx, step.AgentID, workerInput)
            if err != nil {
                return nil, fmt.Errorf("worker %s failed: %w", step.ID, err)
            }
            
            workerResults[step.ID] = result
        }
    }
    
    return workerResults, nil
}

// executeEvaluatorOptimizer executes an evaluator-optimizer workflow
func (wc *WorkflowContext) executeEvaluatorOptimizer(ctx context.Context) (map[string]interface{}, error) {
    if len(wc.Workflow.Steps) < 3 {
        return nil, errors.New("evaluator-optimizer workflow must have at least three steps")
    }
    
    maxIterations := 5 // Configure max iterations to prevent infinite loops
    bestScore := 0.0
    var bestResult map[string]interface{}
    
    for i := 0; i < maxIterations; i++ {
        // Execute generator step
        generatorStep := wc.Workflow.Steps[0]
        wc.CurrentStep = generatorStep.ID
        wc.Visited[generatorStep.ID] = true
        
        result, err := wc.executeStep(ctx, generatorStep)
        if err != nil {
            return nil, fmt.Errorf("generator step failed: %w", err)
        }
        
        // Execute evaluator step
        evaluatorStep := wc.Workflow.Steps[1]
        wc.CurrentStep = evaluatorStep.ID
        wc.Visited[evaluatorStep.ID] = true
        
        evaluatorInput := wc.prepareTaskInput(map[string]interface{}{
            "candidate": result,
        })
        
        evaluation, err := wc.executeTask(ctx, evaluatorStep.AgentID, evaluatorInput)
        if err != nil {
            return nil, fmt.Errorf("evaluator step failed: %w", err)
        }
        
        // Check if we have a new best result
        score, ok := evaluation["score"].(float64)
        if !ok {
            return nil, errors.New("evaluator must return a 'score' number")
        }
        
        if score > bestScore {
            bestScore = score
            bestResult = result
            
            // Execute optimizer step
            optimizerStep := wc.Workflow.Steps[2]
            wc.CurrentStep = optimizerStep.ID
            wc.Visited[optimizerStep.ID] = true
            
            optimizerInput := wc.prepareTaskInput(map[string]interface{}{
                "current_best": result,
                "score":       score,
            })
            
            _, err := wc.executeTask(ctx, optimizerStep.AgentID, optimizerInput)
            if err != nil {
                return nil, fmt.Errorf("optimizer step failed: %w", err)
            }
        }
        
        // Check if we've reached our target score
        if bestScore >= 0.95 { // Configure target score
            break
        }
    }
    
    return bestResult, nil
}
