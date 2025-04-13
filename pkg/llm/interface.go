package llm

import (
    "context"
    "github.com/traddoo/go-agent-framework/pkg/tools"
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