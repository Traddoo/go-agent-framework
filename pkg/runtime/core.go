package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/traddoo/go-agent-framework/pkg/agent"
	"github.com/traddoo/go-agent-framework/pkg/comms"
	"github.com/traddoo/go-agent-framework/pkg/memory"
	"github.com/traddoo/go-agent-framework/pkg/monitoring"
	"github.com/traddoo/go-agent-framework/pkg/tools"
	"github.com/traddoo/go-agent-framework/pkg/workflow"
)

// Runtime represents the core system that coordinates agents and resources
type Runtime struct {
	Config      *Config
	AgentMgr    *AgentManager
	MemoryStore memory.Interface
	CommBus     comms.Bus
	WorkflowEng *workflow.Engine
	ToolReg     tools.Registry
	Monitor     monitoring.System
	Scheduler   *Scheduler
	
	// Internal state
	isRunning   bool
	mu          sync.RWMutex
	// Add resource tracking
	resourceUsage struct {
		llmCalls   atomic.Int64
		memoryUsed atomic.Int64
	}
	// Add cleanup hooks
	cleanupHooks []func() error
}

// Config holds runtime configuration
type Config struct {
	MaxAgents         int
	DefaultLLMConfig  map[string]interface{}
	ResourceLimits    ResourceLimits
	LogLevel          string
	Persistence       bool
	PersistencePath   string
}

// ResourceLimits defines constraints for resource usage
type ResourceLimits struct {
	MaxConcurrentLLMCalls int
	MaxMemoryUsagePerAgent int64
	TokenBudgetPerMinute   int
}

// Task represents a unit of work for an agent
type Task struct {
	ID          string
	Description string
	Input       map[string]interface{}
	Workflow    *workflow.Workflow  // Optional workflow definition
	Deadline    time.Time
	Priority    int
	AssignedTo  string              // Agent or Team ID
	CreatedAt   time.Time
}

// TaskStatus represents the current status of a task
type TaskStatus struct {
	ID          string
	State       string  // "pending", "running", "completed", "failed"
	Progress    float64
	Result      map[string]interface{}
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

// New creates a new runtime instance
func New(cfg *Config) (*Runtime, error) {
	if cfg == nil {
		cfg = &Config{
			MaxAgents: 10,
			ResourceLimits: ResourceLimits{
				MaxConcurrentLLMCalls: 5,
				MaxMemoryUsagePerAgent: 1 << 20, // 1MB
				TokenBudgetPerMinute:   3000,
			},
			LogLevel: "info",
		}
	}
	
	runtime := &Runtime{
		Config: cfg,
	}
	
	// Initialize components
	// These would be fully implemented in their respective files
	runtime.AgentMgr = NewAgentManager(runtime)
	runtime.Scheduler = NewScheduler(runtime)
	
	return runtime, nil
}

// Start initializes and starts the runtime
func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.isRunning {
		return nil
	}
	
	// Initialize and start all components
	if r.Scheduler != nil {
		if err := r.Scheduler.Start(ctx); err != nil {
			return err
		}
	}
	
	// Initialize workflow engine if not already initialized
	if r.WorkflowEng == nil {
		r.WorkflowEng = workflow.NewEngine()
	}
	
	// Initialize other components here if needed
	// This would start memory, comms, monitoring, etc.
	
	r.isRunning = true
	return nil
}

// Stop gracefully shuts down the runtime
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Stop accepting new tasks
	r.Scheduler.Stop(ctx)

	// Wait for running tasks to complete
	done := make(chan struct{})
	go func() {
		r.AgentMgr.WaitForRunningTasks()
		close(done)
	}()

	select {
	case <-shutdownCtx.Done():
		return errors.New("shutdown timeout")
	case <-done:
		return r.cleanup()
	}
}

// CreateAgent creates a new agent with the given configuration
func (r *Runtime) CreateAgent(ctx context.Context, cfg *agent.Config) (*agent.Agent, error) {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	
	// Create a new agent
	agent, err := agent.NewAgent(ctx, cfg)  // Changed from agent.New to agent.NewAgent
	if err != nil {
		return nil, err
	}
	
	// Register the agent with the manager
	if err := r.AgentMgr.RegisterAgent(ctx, agent); err != nil {
		return nil, err
	}
	
	return agent, nil
}

// CreateTeam creates a team of agents
func (r *Runtime) CreateTeam(ctx context.Context, cfg *agent.TeamConfig) (*agent.Team, error) {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	
	// Create a new team
	team, err := agent.NewTeam(cfg)
	if err != nil {
		return nil, err
	}
	
	// Register the team with the manager
	if err := r.AgentMgr.RegisterTeam(ctx, team); err != nil {
		return nil, err
	}
	
	return team, nil
}

// SubmitTask submits a task to be executed by an agent or team
func (r *Runtime) SubmitTask(ctx context.Context, task *Task, targetID string) (string, error) {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	task.CreatedAt = time.Now()
	task.AssignedTo = targetID
	
	// Check if this is a workflow task
	if task.Workflow != nil && r.WorkflowEng != nil {
		// Register the workflow if not already registered
		_, err := r.WorkflowEng.GetWorkflow(task.Workflow.ID)
		if err != nil {
			if err := r.WorkflowEng.RegisterWorkflow(task.Workflow); err != nil {
				return "", fmt.Errorf("failed to register workflow: %w", err)
			}
		}
		
		// Create a context with all necessary dependencies for the workflow
		workflowCtx := context.WithValue(ctx, "agent_manager", r.AgentMgr)
		workflowCtx = context.WithValue(workflowCtx, "tool_registry", r.ToolReg)
		workflowCtx = context.WithValue(workflowCtx, "workflow_engine", r.WorkflowEng)
		
		// Create an agent executor function
		agentExecutor := func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
			agent, err := r.AgentMgr.GetAgent(ctx, agentID)
			if err != nil {
				return nil, fmt.Errorf("agent not found: %w", err)
			}
			
			// Create a task for the agent
			agentTask := map[string]interface{}{
				"Description": fmt.Sprintf("Execute workflow step: %s", task.Workflow.ID),
				"Input":       input,
			}
			
			// Special handling for orchestrator workflows
			if task.Workflow.Type == "orchestrator_workers" {
				// If this is the orchestrator agent
				orchestratorStep := task.Workflow.Steps[task.Workflow.OrchestratorStep]
				if orchestratorStep.AgentID == agentID {
					// Create mock worker assignments
					workerAssignments := make(map[string]interface{})
					
					// For each non-orchestrator step, create an assignment
					for i, step := range task.Workflow.Steps {
						if i != task.Workflow.OrchestratorStep {
							workerAssignments[step.ID] = map[string]interface{}{
								"task": fmt.Sprintf("Task for %s", step.Name),
								"input": input,
								"agent_id": step.AgentID,
							}
						}
					}
					
					// Return a fake orchestrator result with assignments
					return map[string]interface{}{
						"worker_assignments": workerAssignments,
						"plan": map[string]interface{}{
							"subtasks": workerAssignments,
						},
					}, nil
				}
			}
			
			// For non-orchestrator agents or other workflow types
			return agent.ExecuteTask(ctx, agentTask)
		}
		workflowCtx = context.WithValue(workflowCtx, "agent_executor", agentExecutor)
		
		// Execute the workflow
		go func() {
			workflowResult, err := r.WorkflowEng.ExecuteWorkflow(workflowCtx, task.Workflow.ID, task.Input)
			if err != nil {
				// Update task status to failed
				r.Scheduler.UpdateTaskStatus(ctx, task.ID, &TaskStatus{
					ID:        task.ID,
					State:     "failed",
					Error:     err.Error(),
					Progress:  0,
				})
			} else {
				// Convert workflow result to map for storage in TaskStatus
				resultMap := map[string]interface{}{
					"workflow_id": workflowResult.WorkflowID,
					"status": map[string]interface{}{
						"state":    workflowResult.GetStatus().State,
						"progress": workflowResult.GetStatus().Progress,
					},
				}
				
				// Update task status to completed
				r.Scheduler.UpdateTaskStatus(ctx, task.ID, &TaskStatus{
					ID:        task.ID,
					State:     "completed",
					Progress:  1.0,
					Result:    resultMap,
				})
			}
		}()
	}
	
	// Submit the task to the scheduler
	if err := r.Scheduler.ScheduleTask(ctx, task); err != nil {
		return "", err
	}
	
	return task.ID, nil
}

// GetTaskStatus retrieves the status of a task
func (r *Runtime) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	// Get task status from the scheduler
	return r.Scheduler.GetTaskStatus(ctx, taskID)
}

func (r *Runtime) cleanup() error {
	var errs []error
	for _, hook := range r.cleanupHooks {
		if err := hook(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ExecuteWorkflow executes a workflow
func (r *Runtime) ExecuteWorkflow(ctx context.Context, wf *workflow.Workflow) (*workflow.Result, error) {
	result := &workflow.Result{
		WorkflowID: wf.Name,
	}
	// Initialize the status properly
	result.SetStatus(workflow.Status{
		State:    "running",
		Progress: 0.0,
	})
	
	// Create a done channel to signal completion
	done := make(chan struct{})
	
	// Execute workflow in a goroutine
	go func() {
		defer close(done)
		
		// Execute each step sequentially
		for _, step := range wf.Steps {
			for _, task := range step.Tasks {
				agent, err := r.AgentMgr.GetAgent(ctx, task.AgentID)
				if err != nil {
					result.SetStatus(workflow.Status{
						State:    "failed",
						Progress: 0.0,
						Error:    err,
					})
					return
				}
				
				// Prepare task in the format expected by the agent
				agentTask := map[string]interface{}{
					"Description": fmt.Sprintf("Execute workflow task for %s", step.Name),
					"Input":       task.Input,
				}
				
				// Execute the task
				_, err = agent.ExecuteTask(ctx, agentTask)
				if err != nil {
					result.SetStatus(workflow.Status{
						State:    "failed",
						Progress: 0.0,
						Error:    err,
					})
					return
				}
			}
		}
		
		// All steps completed successfully
		result.SetStatus(workflow.Status{
			State:    "completed",
			Progress: 1.0,
		})
	}()
	
	// Don't wait here, just return the result object that can be queried for status
	return result, nil
}