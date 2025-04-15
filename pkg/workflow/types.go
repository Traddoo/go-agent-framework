package workflow

// Step represents a workflow step
type Step struct {
    ID           string
    Name         string
    Type         string  // "agent", "llm", "tool", "conditional", "subworkflow"
    AgentID      string  // For "agent" steps
    PromptTemplate string  // For "llm" steps
    ToolName     string  // For "tool" steps
    Condition    string  // For "conditional" steps
    WorkflowID   string  // For "subworkflow" steps
    NextSteps    []string  // IDs of next steps
    Tasks        []Task    // Tasks to be executed in this step
    Parallel     bool      // Whether tasks can be executed in parallel
}

// Task represents a workflow task
type Task struct {
    AgentID string
    Input   map[string]interface{}
}

// Workflow represents a workflow definition
type Workflow struct {
    ID            string
    Name          string
    Description   string
    Type          string  // "prompt_chaining", "routing", "parallelization", "orchestrator_workers", "evaluator_optimizer"
    Steps         []Step
    
    // Type-specific configurations
    RouterStepIndex  int                    // For "routing" workflows
    ParallelGroups   map[string][]int       // For "parallelization" workflows
    OrchestratorStep int                    // For "orchestrator_workers" workflows
    EvaluatorStep    int                    // For "evaluator_optimizer" workflows
}

// Status represents the current status of a workflow
type Status struct {
    State    string
    Progress float64
    Error    error
}

// Result represents the result of a workflow execution
type Result struct {
    WorkflowID string
    status     Status
}

// GetStatus returns the current status of the workflow
func (r *Result) GetStatus() Status {
    return r.status
}

// SetStatus sets the status of the workflow
func (r *Result) SetStatus(status Status) {
    r.status = status
}
