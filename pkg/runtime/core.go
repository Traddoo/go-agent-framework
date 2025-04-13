package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/thomasfaulds/go-agent-framework/pkg/agent"
	"github.com/thomasfaulds/go-agent-framework/pkg/comms"
	"github.com/thomasfaulds/go-agent-framework/pkg/memory"
	"github.com/thomasfaulds/go-agent-framework/pkg/monitoring"
	"github.com/thomasfaulds/go-agent-framework/pkg/tools"
	"github.com/thomasfaulds/go-agent-framework/pkg/workflow"
)

// Runtime represents the core system that coordinates agents and resources
type Runtime struct {
	Config      *Config
	AgentMgr    *AgentManager
	MemoryStore memory.Interface
	CommBus     comms.Bus
	WorkflowEng workflow.Engine
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
	r.Scheduler.Stop()

	// Wait for running tasks
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
	agent, err := agent.New(cfg)
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
