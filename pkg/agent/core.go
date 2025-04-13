package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"encoding/json"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/thomasfaulds/go-agent-framework/pkg/llm"
	"github.com/thomasfaulds/go-agent-framework/pkg/memory"
	"github.com/thomasfaulds/go-agent-framework/pkg/tools"
)

// AgentError represents a structured error for the agent
type AgentError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AgentError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Agent represents a single agent with LLM capabilities
type Agent struct {
	ID          string
	Name        string
	Description string
	LLM         llm.Interface
	Tools       []tools.Tool
	Memory      memory.Interface
	TaskQueue   []*Task
	Config      *Config
	State       *State
	
	mutex       sync.RWMutex
	metrics     struct {
		tasksProcessed    atomic.Int64
		taskErrors        atomic.Int64
		avgProcessingTime atomic.Float64
	}
}

// Config holds agent configuration
type Config struct {
	ID                 string
	Name               string
	Description        string
	MaxConcurrentTasks int
	DefaultSystemPrompt string
	ModelParameters    map[string]interface{}
	LLMProvider        string  // "anthropic" or "openai"
	LLMModel           string  // specific model name
}

// State holds agent state information
type State struct {
	CurrentTasks    map[string]*Task
	KnowledgeBase   *memory.Chunk
	mutex           sync.RWMutex
}

// Task represents a task for an agent to execute
type Task struct {
	ID          string
	Description string
	Input       map[string]interface{}
	Status      string
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	Result      map[string]interface{}
	Error       string
}

// New creates a new agent with the given configuration
func New(cfg *Config) (*Agent, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	
	if cfg.Name == "" {
		cfg.Name = "Agent-" + cfg.ID[:8]
	}
	
	if cfg.MaxConcurrentTasks <= 0 {
		cfg.MaxConcurrentTasks = 1
	}
	
	// Create a default system prompt if not provided
	if cfg.DefaultSystemPrompt == "" {
		cfg.DefaultSystemPrompt = "You are a helpful AI assistant named " + cfg.Name + "."
	}
	
	// Initialize the LLM based on the provider
	var llmInterface llm.Interface
	var err error
	
	switch cfg.LLMProvider {
	case "anthropic":
		llmInterface, err = llm.NewAnthropicLLM(cfg.LLMModel, cfg.ModelParameters)
	case "openai":
		llmInterface, err = llm.NewOpenAILLM(cfg.LLMModel, cfg.ModelParameters)
	default:
		return nil, errors.New("unsupported LLM provider")
	}
	
	if err != nil {
		return nil, err
	}
	
	// Create a new agent
	agent := &Agent{
		ID:          cfg.ID,
		Name:        cfg.Name,
		Description: cfg.Description,
		LLM:         llmInterface,
		Tools:       make([]tools.Tool, 0),
		TaskQueue:   make([]*Task, 0),
		Config:      cfg,
		State: &State{
			CurrentTasks: make(map[string]*Task),
		},
	}
	
	return agent, nil
}

// RegisterTool registers a tool with the agent
func (a *Agent) RegisterTool(tool tools.Tool) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	// Add the tool to the agent's tool registry
	a.Tools = append(a.Tools, tool)
	
	return nil
}

// ExecuteTask executes a task
func (a *Agent) ExecuteTask(ctx context.Context, task interface{}) (map[string]interface{}, error) {
	if task == nil {
		return nil, &AgentError{
			Code:    "INVALID_TASK",
			Message: "task cannot be nil",
		}
	}
	a.mutex.Lock()
	
	// Convert the task interface to our internal task type
	// In practice, you'd want to handle this conversion more robustly
	taskData, ok := task.(map[string]interface{})
	if !ok {
		a.mutex.Unlock()
		return nil, errors.New("invalid task format")
	}
	
	// Create an internal task
	newTask := &Task{
		ID:          uuid.New().String(),
		Description: taskData["Description"].(string),
		Input:       taskData["Input"].(map[string]interface{}),
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	
	// Check if we have capacity for this task
	if len(a.State.CurrentTasks) >= a.Config.MaxConcurrentTasks {
		// Add to queue if we're at capacity
		a.TaskQueue = append(a.TaskQueue, newTask)
		a.mutex.Unlock()
		return nil, errors.New("task queued for execution")
	}
	
	// Add the task to current tasks
	a.State.CurrentTasks[newTask.ID] = newTask
	a.mutex.Unlock()
	
	// Now process the task asynchronously
	go a.processTask(ctx, newTask)
	
	// Return that the task is being processed
	return map[string]interface{}{
		"status": "processing",
		"taskId": newTask.ID,
	}, nil
}

// GetTaskStatus retrieves the status of a task
func (a *Agent) GetTaskStatus(ctx context.Context, taskID string) (*Task, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	
	// Check if the task exists
	task, exists := a.State.CurrentTasks[taskID]
	if !exists {
		return nil, errors.New("task not found")
	}
	
	return task, nil
}

// processTask processes a task using the agent's LLM and tools
func (a *Agent) processTask(ctx context.Context, task *Task) {
	// Mark the task as running
	a.mutex.Lock()
	task.Status = "running"
	task.StartedAt = time.Now()
	a.mutex.Unlock()
	
	// Create the prompt for the LLM
	prompt := a.Config.DefaultSystemPrompt + "\n\n"
	prompt += "Task: " + task.Description + "\n"
	prompt += "Input: " + formatTaskInput(task.Input) + "\n"
	
	// Call the LLM with the prompt and tools
	response, err := a.LLM.CompleteWithTools(ctx, prompt, a.Tools, a.Config.ModelParameters)
	
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	// Update the task with the result or error
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Result = map[string]interface{}{
			"response": response,
		}
	}
	
	task.CompletedAt = time.Now()
	
	// Check if there are any tasks in the queue and process the next one
	if len(a.TaskQueue) > 0 {
		nextTask := a.TaskQueue[0]
		a.TaskQueue = a.TaskQueue[1:]
		a.State.CurrentTasks[nextTask.ID] = nextTask
		
		go a.processTask(ctx, nextTask)
	}
}

// formatTaskInput formats the task input for the prompt
func formatTaskInput(input map[string]interface{}) string {
	// Convert the input to a JSON string
	jsonBytes, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return "Error formatting input: " + err.Error()
	}
	
	return string(jsonBytes)
}
