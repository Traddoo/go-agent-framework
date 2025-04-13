package tools

import (
	"context"
)

// Tool defines the interface for tools that can be used by agents
type Tool interface {
	// Name returns the name of the tool
	Name() string
	
	// Description returns a description of the tool
	Description() string
	
	// Parameters returns a map of parameter names to their JSON schema definitions
	Parameters() interface{}
	
	// Execute executes the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// Registry holds all available tools
type Registry interface {
	// RegisterTool registers a tool with the registry
	RegisterTool(tool Tool) error
	
	// GetTool retrieves a tool by name
	GetTool(name string) (Tool, error)
	
	// ListTools lists all registered tools
	ListTools() []Tool
}

// ToolRegistry is the default implementation of Registry
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool registers a tool with the registry
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	r.tools[tool.Name()] = tool
	return nil
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, ErrToolNotFound
	}
	return tool, nil
}

// ListTools lists all registered tools
func (r *ToolRegistry) ListTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ErrToolNotFound is returned when a tool is not found
var ErrToolNotFound = ToolError{
	Code:    "tool_not_found",
	Message: "Tool not found",
}

// ToolError represents an error when using a tool
type ToolError struct {
	Code    string
	Message string
}

// Error returns the error message
func (e ToolError) Error() string {
	return e.Message
}
