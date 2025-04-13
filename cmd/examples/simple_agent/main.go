package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/thomasfaulds/go-agent-framework/pkg/agent"
	"github.com/thomasfaulds/go-agent-framework/pkg/memory"
	"github.com/thomasfaulds/go-agent-framework/pkg/runtime"
	"github.com/thomasfaulds/go-agent-framework/pkg/tools"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create the runtime
	rt, err := runtime.New(&runtime.Config{
		MaxAgents:         10,
		DefaultLLMConfig:  map[string]interface{}{"temperature": 0.7},
		ResourceLimits:    runtime.ResourceLimits{MaxConcurrentLLMCalls: 5},
		LogLevel:          "info",
	})
	if err != nil {
		log.Fatalf("Failed to create runtime: %v", err)
	}

	// Start the runtime
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop(ctx)

	// Register some standard tools
	toolRegistry := tools.NewToolRegistry()
	registerStandardTools(toolRegistry)

	// Create an in-memory document store
	docStore := memory.NewInMemoryStore()
	docStore.Initialize(ctx)

	// Create an agent
	agentConfig := &agent.Config{
		Name:               "AssistantAgent",
		Description:        "A helpful assistant agent",
		MaxConcurrentTasks: 3,
		DefaultSystemPrompt: "You are a helpful AI assistant named AssistantAgent. You help users by answering questions and performing tasks.",
		LLMProvider:        "anthropic",
		LLMModel:           "claude-3-5-sonnet",
		ModelParameters:    map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  4096,
		},
	}

	assistantAgent, err := rt.CreateAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create a task for the agent
	task := &runtime.Task{
		Description: "Answer the user's question about artificial intelligence",
		Input: map[string]interface{}{
			"question": "What are the key components of a multi-agent system?",
		},
		Priority: 1,
	}

	// Submit the task to the agent
	taskID, err := rt.SubmitTask(ctx, task, assistantAgent.ID)
	if err != nil {
		log.Fatalf("Failed to submit task: %v", err)
	}

	log.Printf("Submitted task with ID: %s", taskID)

	// Poll for task completion
	for {
		status, err := rt.GetTaskStatus(ctx, taskID)
		if err != nil {
			log.Fatalf("Failed to get task status: %v", err)
		}

		log.Printf("Task status: %s, Progress: %.2f", status.State, status.Progress)

		if status.State == "completed" || status.State == "failed" {
			if status.State == "completed" {
				log.Printf("Task completed successfully. Result: %v", status.Result)
			} else {
				log.Printf("Task failed with error: %s", status.Error)
			}
			break
		}

		time.Sleep(1 * time.Second)
	}
}

// registerStandardTools registers standard tools with the registry
func registerStandardTools(registry *tools.ToolRegistry) {
	// Register a simple calculator tool
	calculatorTool := &CalculatorTool{}
	registry.RegisterTool(calculatorTool)

	// Register a weather tool
	weatherTool := &WeatherTool{}
	registry.RegisterTool(weatherTool)
}

// CalculatorTool is a simple calculator tool
type CalculatorTool struct{}

// Name returns the name of the tool
func (t *CalculatorTool) Name() string {
	return "calculator"
}

// Description returns a description of the tool
func (t *CalculatorTool) Description() string {
	return "A simple calculator that can perform basic arithmetic operations"
}

// Parameters returns the parameters for the tool
func (t *CalculatorTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{"add", "subtract", "multiply", "divide"},
				"description": "The operation to perform",
			},
			"a": map[string]interface{}{
				"type": "number",
				"description": "The first operand",
			},
			"b": map[string]interface{}{
				"type": "number",
				"description": "The second operand",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

// Execute executes the tool
func (t *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	operation := args["operation"].(string)
	a := args["a"].(float64)
	b := args["b"].(float64)

	// Perform the operation
	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, tools.ToolError{
				Code:    "division_by_zero",
				Message: "Division by zero is not allowed",
			}
		}
		result = a / b
	default:
		return nil, tools.ToolError{
			Code:    "invalid_operation",
			Message: "Invalid operation: " + operation,
		}
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

// WeatherTool is a tool for getting weather information
type WeatherTool struct{}

// Name returns the name of the tool
func (t *WeatherTool) Name() string {
	return "weather"
}

// Description returns a description of the tool
func (t *WeatherTool) Description() string {
	return "Get current weather information for a location"
}

// Parameters returns the parameters for the tool
func (t *WeatherTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type": "string",
				"description": "The location to get weather information for (city name or coordinates)",
			},
			"units": map[string]interface{}{
				"type": "string",
				"enum": []string{"metric", "imperial"},
				"default": "metric",
				"description": "The units to use for temperature (metric: Celsius, imperial: Fahrenheit)",
			},
		},
		"required": []string{"location"},
	}
}

// Execute executes the tool
func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	location := args["location"].(string)
	
	// Get units (default to metric)
	units := "metric"
	if unitsArg, ok := args["units"]; ok {
		units = unitsArg.(string)
	}

	// In a real implementation, this would call a weather API
	// For this example, we'll just return mock data
	return map[string]interface{}{
		"location": location,
		"temperature": 22.5,
		"units": units,
		"condition": "partly cloudy",
		"humidity": 65,
		"wind_speed": 10.2,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}
