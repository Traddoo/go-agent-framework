package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/traddoo/go-agent-framework/pkg/tools"
)

// AnthropicLLM implements the Interface for Anthropic's Claude models
type AnthropicLLM struct {
	APIKey      string
	Model       string
	Params      map[string]interface{}
	Client      anthropic.Client  // Changed: Remove pointer, store as value
	initialized bool
}

// Initialize initializes the Anthropic client
func (a *AnthropicLLM) Initialize(ctx context.Context) error {
	// Get API key from environment variable if not set
	if a.APIKey == "" {
		a.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		if a.APIKey == "" {
			return errors.New("ANTHROPIC_API_KEY environment variable not set")
		}
	}
	
	// Set default model if not set
	if a.Model == "" {
		a.Model = "claude-3-sonnet-20240229"
	}
	
	// Store the client as a value
	a.Client = anthropic.NewClient(option.WithAPIKey(a.APIKey))
	a.initialized = true
	
	return nil
}

// Complete generates a completion using Anthropic's Claude
func (a *AnthropicLLM) Complete(ctx context.Context, prompt string, params map[string]interface{}) (*Response, error) {
	// Check if client is initialized
	if !a.initialized {
		if err := a.Initialize(ctx); err != nil {
			return nil, err
		}
	}
	
	// Merge default params with provided params
	mergedParams := make(map[string]interface{})
	for k, v := range a.Params {
		mergedParams[k] = v
	}
	for k, v := range params {
		mergedParams[k] = v
	}
	
	// Extract parameters
	temp := 0.7
	if tempParam, ok := mergedParams["temperature"]; ok {
		if tempFloat, ok := tempParam.(float64); ok {
			temp = tempFloat
		}
	}
	
	maxTokens := int64(4096)
	if mt, ok := mergedParams["max_tokens"]; ok {
		if mtInt, ok := mt.(int); ok {
			maxTokens = int64(mtInt)
		}
	}
	
	// Create a system prompt if provided
	systemPrompt := ""
	if sys, ok := mergedParams["system"]; ok {
		if sysStr, ok := sys.(string); ok {
			systemPrompt = sysStr
		}
	}
	
	// Create the message request
	messages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(prompt),
			},
		},
	}
	
	// Create the request parameters
	req := anthropic.MessageNewParams{
		Model:       a.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temp),
	}
	
	// Add system prompt if provided
	if systemPrompt != "" {
		req.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}
	
	// Make the API call
	resp, err := a.Client.Messages.New(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API error: %w", err)
	}
	
	// Extract the text content from the response
	var content string
	for _, block := range resp.Content {
		if blockObj, ok := block.AsAny().(anthropic.TextBlock); ok {
			content += blockObj.Text
		}
	}
	
	// Build our response
	response := &Response{
		Content: content,
		Usage: Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
		FinishReason: string(resp.StopReason),
	}
	
	return response, nil
}

// CompleteWithTools generates a completion with tool usage
func (a *AnthropicLLM) CompleteWithTools(ctx context.Context, prompt string, toolsList []tools.Tool, params map[string]interface{}) (*Response, error) {
	// Check if client is initialized
	if !a.initialized {
		fmt.Printf("[ANTHROPIC] Initializing client with model: %s\n", a.Model)
		if err := a.Initialize(ctx); err != nil {
			fmt.Printf("[ANTHROPIC] Initialization error: %v\n", err)
			return nil, err
		}
		fmt.Printf("[ANTHROPIC] Client initialized successfully\n")
	}
	
	// Merge default params with provided params
	mergedParams := make(map[string]interface{})
	for k, v := range a.Params {
		mergedParams[k] = v
	}
	for k, v := range params {
		mergedParams[k] = v
	}
	
	// Extract parameters
	temp := 0.7
	if tempParam, ok := mergedParams["temperature"]; ok {
		if tempFloat, ok := tempParam.(float64); ok {
			temp = tempFloat
		}
	}
	
	maxTokens := int64(4096)
	if mt, ok := mergedParams["max_tokens"]; ok {
		if mtInt, ok := mt.(int); ok {
			maxTokens = int64(mtInt)
		}
	}
	
	// Create a system prompt if provided
	systemPrompt := "You are a helpful AI assistant that can use tools."
	if sys, ok := mergedParams["system"]; ok {
		if sysStr, ok := sys.(string); ok {
			systemPrompt = sysStr
		}
	}
	
	// Convert tools to Anthropic tool format
	anthropicTools := make([]anthropic.ToolUnionParam, 0, len(toolsList))
	for _, tool := range toolsList {
		// Create the tool input schema
		toolSchema := extractToolSchema(tool)
		
		// Create the tool
		toolParam := anthropic.ToolParam{
			Name:        tool.Name(),
			Description: anthropic.String(tool.Description()),
			InputSchema: toolSchema,
		}
		
		// Create the tool union
		anthropicTool := anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
		
		anthropicTools = append(anthropicTools, anthropicTool)
	}
	
	// Create the user message
	messages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(prompt),
			},
		},
	}
	
	// Create the request parameters
	req := anthropic.MessageNewParams{
		Model:       a.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temp),
		Tools:       anthropicTools,
	}
	
	// Add system prompt
	req.System = []anthropic.TextBlockParam{
		{Text: systemPrompt},
	}
	
	// Log before making API call
	fmt.Printf("[ANTHROPIC] Making API call to Anthropic, model: %s, tools: %d\n", a.Model, len(anthropicTools))
	
	// Make the API call
	resp, err := a.Client.Messages.New(ctx, req)
	if err != nil {
		fmt.Printf("[ANTHROPIC] API error: %v\n", err)
		return nil, fmt.Errorf("Anthropic API error: %w", err)
	}
	
	fmt.Printf("[ANTHROPIC] API call successful, response received\n")
	
	// Process tool calls
	toolCalls := []ToolCall{}
	
	// Process tool use blocks and collect results
	toolUseBlocks := []anthropic.ToolUseBlock{}
	for _, block := range resp.Content {
		if blockObj, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			toolUseBlocks = append(toolUseBlocks, blockObj)
		}
	}
	
	// If there are tool use blocks, handle them
	if len(toolUseBlocks) > 0 {
		// Create a map of tools by name for quick lookup
		toolMap := make(map[string]interface{})
		for _, tool := range toolsList {
			toolMap[tool.Name()] = tool
		}
		
		// Process each tool use
		for _, toolUse := range toolUseBlocks {
			// Find the matching tool
			toolInterface, exists := toolMap[toolUse.Name]
			if !exists {
				continue // Skip if tool not found
			}
			
			tool, ok := toolInterface.(tools.Tool)
			if !ok {
				continue // Skip if not a valid tool
			}
			
			// Parse the arguments
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &args); err != nil {
				continue // Skip if can't parse arguments
			}
			
			// Execute the tool
			result, err := tool.Execute(ctx, args)
			if err != nil {
				// Record error result
				toolCalls = append(toolCalls, ToolCall{
					ToolName:  toolUse.Name,
					Arguments: args,
					Response:  fmt.Sprintf("Error: %s", err.Error()),
				})
			} else {
				// Record successful result
				toolCalls = append(toolCalls, ToolCall{
					ToolName:  toolUse.Name,
					Arguments: args,
					Response:  result,
				})
			}
		}
	}
	
	// Extract the text content from the response
	var content string
	for _, block := range resp.Content {
		if blockObj, ok := block.AsAny().(anthropic.TextBlock); ok {
			content += blockObj.Text
		}
	}
	
	// Build the final response
	response := &Response{
		Content:      content,
		ToolCalls:    toolCalls,
		Usage: Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
		FinishReason: string(resp.StopReason),
	}
	
	return response, nil
}

// extractToolSchema extracts the JSON schema for a tool
func extractToolSchema(tool tools.Tool) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{}
	
	// Get the parameters from the tool
	params := tool.Parameters()
	
	// Convert to a map
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		// Default to object type if conversion fails
		schema.Properties = make(map[string]interface{})
		return schema
	}
	
	// Create a new map for the schema
	schemaMap := make(map[string]interface{})
	
	// Set type (default to "object" if not specified)
	schemaType := "object"
	if typeVal, ok := paramsMap["type"].(string); ok {
		schemaType = typeVal
	}
	schemaMap["type"] = schemaType
	
	// Extract properties
	if props, ok := paramsMap["properties"].(map[string]interface{}); ok {
		schema.Properties = props
	}
	
	// Extract required fields if present
	if required, ok := paramsMap["required"].([]string); ok {
		schema.ExtraFields = map[string]interface{}{
			"required": required,
		}
	} else if reqArray, ok := paramsMap["required"].([]interface{}); ok {
		required := make([]string, 0, len(reqArray))
		for _, req := range reqArray {
			if reqStr, ok := req.(string); ok {
				required = append(required, reqStr)
			}
		}
		if len(required) > 0 {
			schema.ExtraFields = map[string]interface{}{
				"required": required,
			}
		}
	}
	
	return schema
}

// Modified to return a proper JSON schema type
func (a *AnthropicLLM) parseToolType(typeStr string) map[string]interface{} {
	return map[string]interface{}{
		"type": typeStr,
	}
}
