package llm

import (
	"context"
	
	"github.com/thomasfaulds/go-agent-framework/pkg/tools"
)

// Interface defines the common interface for LLM providers
type Interface interface {
	// Initialize initializes the LLM provider
	Initialize(ctx context.Context) error
	
	// Complete generates a completion for the given prompt
	Complete(ctx context.Context, prompt string, params map[string]interface{}) (*Response, error)
	
	// CompleteWithTools generates a completion with tool usage
	CompleteWithTools(ctx context.Context, prompt string, tools []tools.Tool, params map[string]interface{}) (*Response, error)
}

// Response represents a response from an LLM
type Response struct {
	Content      string
	ToolCalls    []ToolCall
	Usage        Usage
	FinishReason string
}

// ToolCall represents a tool call from an LLM
type ToolCall struct {
	ToolName    string
	Arguments   map[string]interface{}
	Response    interface{}
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewAnthropicLLM creates a new Anthropic LLM client
func NewAnthropicLLM(model string, params map[string]interface{}) (Interface, error) {
	return &AnthropicLLM{
		Model: model,
		Params: params,
	}, nil
}

// NewOpenAILLM creates a new OpenAI LLM client
func NewOpenAILLM(model string, params map[string]interface{}) (Interface, error) {
	return &OpenAILLM{
		Model: model,
		Params: params,
	}, nil
}
