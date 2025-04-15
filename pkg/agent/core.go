package agent

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"
    
    "github.com/google/uuid"
    "github.com/traddoo/go-agent-framework/pkg/llm"
    "github.com/traddoo/go-agent-framework/pkg/tools"
)

func (e *AgentError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAgent creates a new agent with the given configuration
func NewAgent(ctx context.Context, cfg *Config) (*Agent, error) {
    if cfg.ID == "" {
        return nil, errors.New("agent ID is required")
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
        Tools:       make(map[string]tools.Tool),
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
    toolName := tool.Name()
    a.Tools[toolName] = tool
    
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
        StartedAt:   time.Now(),
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

// processTask processes a task using the agent's LLM and tools
func (a *Agent) processTask(ctx context.Context, task *Task) (map[string]interface{}, error) {
    // Mark the task as running
    a.mutex.Lock()
    task.Status = "running"
    task.StartedAt = time.Now()
    a.mutex.Unlock()

    fmt.Printf("[AGENT] %s: Processing task: %s\n", a.Config.Name, task.Description)

    // Create the prompt for the LLM based on the agent's role
    prompt := a.Config.DefaultSystemPrompt + "\n\n"
    prompt += "Task: " + task.Description + "\n"
    prompt += "Input: " + formatTaskInput(task.Input) + "\n"

    // Log available tools
    fmt.Printf("[AGENT] %s: Available tools (%d): ", a.Config.Name, len(a.Tools))
    toolNames := []string{}
    for toolName := range a.Tools {
        toolNames = append(toolNames, toolName)
    }
    fmt.Printf("%v\n", toolNames)

    // Convert tools map to slice for LLM interface
    toolsList := make([]tools.Tool, 0, len(a.Tools))
    for _, tool := range a.Tools {
        toolsList = append(toolsList, tool)
    }

    fmt.Printf("[AGENT] %s: Calling LLM with prompt length: %d characters\n", a.Config.Name, len(prompt))
    
    // Add a special hint in the prompt to use tools
    prompt += "\n\nIMPORTANT: Use the available tools to complete this task. You have access to: " + 
              strings.Join(toolNames, ", ") + ".\n"

    // Call the LLM with the prompt and tools
    response, err := a.LLM.CompleteWithTools(ctx, prompt, toolsList, a.Config.ModelParameters)
    if err != nil {
        fmt.Printf("[AGENT] %s: LLM error: %v\n", a.Config.Name, err)
        return nil, fmt.Errorf("LLM completion failed: %w", err)
    }

    fmt.Printf("[AGENT] %s: LLM response received, length: %d, tool calls: %d\n", 
               a.Config.Name, len(response.Content), len(response.ToolCalls))

    // Process any tool calls from the response
    result := make(map[string]interface{})
    if len(response.ToolCalls) == 0 {
        fmt.Printf("[AGENT] %s: WARNING - No tool calls made by the agent\n", a.Config.Name)
    }
    
    for i, toolCall := range response.ToolCalls {
        fmt.Printf("[AGENT] %s: Executing tool call %d: %s with args: %v\n", 
                   a.Config.Name, i+1, toolCall.ToolName, toolCall.Arguments)
        
        toolResult, err := a.executeTool(ctx, toolCall.ToolName, toolCall.Arguments)
        if err != nil {
            fmt.Printf("[AGENT] %s: Tool execution failed: %v\n", a.Config.Name, err)
            return nil, fmt.Errorf("tool execution failed: %w", err)
        }
        
        // Print detailed tool results for debugging
        fmt.Printf("[AGENT] %s: Tool execution successful. Result: %v\n", a.Config.Name, toolResult)
        result[toolCall.ToolName] = toolResult
        
        // Special handling for document_update tool to show content
        if toolCall.ToolName == "document_update" {
            if content, ok := toolCall.Arguments["content"].(string); ok {
                fmt.Printf("\n==== DOCUMENT UPDATE BY %s ====\n%s\n==== END UPDATE ====\n\n", 
                          a.Config.Name, content)
            }
        }
        
        // Special handling for brave_search to show what was found
        if toolCall.ToolName == "brave_search" {
            fmt.Printf("\n==== SEARCH RESULTS FOR %s ====\n", a.Config.Name)
            
            if results, ok := toolResult.(map[string]interface{}); ok {
                if searchResults, ok := results["results"].([]map[string]interface{}); ok {
                    for i, r := range searchResults {
                        fmt.Printf("%d. %s\n   %s\n", i+1, r["title"], r["snippet"])
                    }
                }
            }
            
            fmt.Printf("==== END SEARCH RESULTS ====\n\n")
        }
    }

    // Add the final response to the result
    result["response"] = response.Content
    
    // Print the LLM's final thoughts
    fmt.Printf("\n==== %s FINAL RESPONSE ====\n%s\n==== END RESPONSE ====\n\n", 
               a.Config.Name, response.Content)

    a.mutex.Lock()
    task.Status = "completed"
    task.CompletedAt = time.Now()
    a.mutex.Unlock()

    fmt.Printf("[AGENT] %s: Task completed in %.2f seconds\n", 
               a.Config.Name, task.CompletedAt.Sub(task.StartedAt).Seconds())

    return result, nil
}

// formatTaskInput formats the task input for the prompt
func formatTaskInput(input map[string]interface{}) string {
    // Simple implementation - in practice you'd want more robust formatting
    return fmt.Sprintf("%v", input)
}

// executeTool executes a specific tool
func (a *Agent) executeTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
    tool, exists := a.Tools[toolName]
    if !exists {
        return nil, fmt.Errorf("tool %s not found", toolName)
    }

    return tool.Execute(ctx, args)
}
