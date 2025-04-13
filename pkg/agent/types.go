package agent

import (
    "sync"
    "sync/atomic"
    "time"

    "github.com/traddoo/go-agent-framework/pkg/llm"
    "github.com/traddoo/go-agent-framework/pkg/memory"
    "github.com/traddoo/go-agent-framework/pkg/tools"
)

// AgentError represents a structured error for the agent
type AgentError struct {
    Code    string
    Message string
    Cause   error
}

// Agent represents a single agent with LLM capabilities
type Agent struct {
    ID          string
    Name        string
    Description string
    LLM         llm.Interface
    Tools       map[string]tools.Tool
    Memory      memory.Interface
    TaskQueue   []*Task
    Config      *Config
    State       *State
    
    mutex       sync.RWMutex
    metrics     struct {
        tasksProcessed    atomic.Int64
        taskErrors        atomic.Int64
        avgProcessingTime uint64
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
    LLMProvider        string
    LLMModel           string
}

// State holds agent state information
type State struct {
    CurrentTasks    map[string]*Task
    KnowledgeBase   *memory.Chunk
    mutex           sync.RWMutex
}

// Task represents a unit of work for an agent
type Task struct {
    ID          string
    Description string
    Input       map[string]interface{}
    Status      string
    StartedAt   time.Time
    CompletedAt time.Time
    Result      map[string]interface{}
    Error       string
}