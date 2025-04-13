package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/traddoo/go-agent-framework/pkg/llm"
	"github.com/traddoo/go-agent-framework/pkg/tools"
)

// Simple calculator tool for testing
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "A simple calculator that can perform basic arithmetic operations"
}

func (t *CalculatorTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
				"description": "The operation to perform",
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "The first operand",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "The second operand",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

func (t *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation must be a string")
	}

	a, ok := args["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}

	b, ok := args["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}

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
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

// Current weather tool for testing
type WeatherTool struct{}

func (t *WeatherTool) Name() string {
	return "get_current_weather"
}

func (t *WeatherTool) Description() string {
	return "Get the current weather in a given location"
}

func (t *WeatherTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
			"unit": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"celsius", "fahrenheit"},
				"description": "The unit of temperature to use",
			},
		},
		"required": []string{"location"},
	}
}

func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, ok := args["location"].(string)
	if !ok {
		return nil, fmt.Errorf("location must be a string")
	}

	unit := "celsius"
	if unitArg, ok := args["unit"].(string); ok {
		unit = unitArg
	}

	// Simulate API call with mock data
	return map[string]interface{}{
		"location":    location,
		"temperature": 22.5,
		"unit":        unit,
		"condition":   "sunny",
		"humidity":    60,
		"timestamp":   time.Now().Format(time.RFC3339),
	}, nil
}

func main() {
	// Create a context
	ctx := context.Background()

	// Create an Anthropic LLM client
	anthropicLLM, err := llm.NewAnthropicLLM("claude-3-5-sonnet-20240620", map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  4096,
	})
	if err != nil {
		fmt.Printf("Error creating Anthropic LLM: %v\n", err)
		os.Exit(1)
	}

	// Initialize the client
	if err := anthropicLLM.Initialize(ctx); err != nil {
		fmt.Printf("Error initializing Anthropic LLM: %v\n", err)
		os.Exit(1)
	}

	// Create some tools
	calculatorTool := &CalculatorTool{}
	weatherTool := &WeatherTool{}

	// Test a basic completion
	fmt.Println("=== Testing Basic Completion ===")
	resp, err := anthropicLLM.Complete(ctx, "What is the capital of France?", nil)
	if err != nil {
		fmt.Printf("Error during completion: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Tokens used: %d\n\n", resp.Usage.TotalTokens)

	// Test completion with tools
	fmt.Println("=== Testing Completion with Tools ===")
	toolsResp, err := anthropicLLM.CompleteWithTools(
		ctx,
		"I need to calculate 25 divided by 5, and then find out the current weather in Boston.",
		[]tools.Tool{calculatorTool, weatherTool},
		map[string]interface{}{
			"system": "You are a helpful assistant that can use tools to answer questions.",
		},
	)
	if err != nil {
		fmt.Printf("Error during tool completion: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Final response: %s\n", toolsResp.Content)
	fmt.Printf("Tool calls made: %d\n", len(toolsResp.ToolCalls))
	
	for i, call := range toolsResp.ToolCalls {
		fmt.Printf("Tool call %d: %s\n", i+1, call.ToolName)
		fmt.Printf("  Arguments: %v\n", call.Arguments)
		fmt.Printf("  Response: %v\n", call.Response)
	}
	
	fmt.Printf("Total tokens used: %d\n", toolsResp.Usage.TotalTokens)
}