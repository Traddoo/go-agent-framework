package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport"
)

// MCPClient wraps the MCP client functionality
type MCPClient struct {
	client         *mcp_golang.Client
	transport      transport.Transport
	availableTools []ToolDefinition
}

// ToolDefinition represents a tool available via MCP
type ToolDefinition struct {
	Name        string
	Description *string
	Parameters  map[string]interface{}
}

// NewMCPClient creates a new MCP client
func NewMCPClient(transport transport.Transport) (*MCPClient, error) {
	client := mcp_golang.NewClient(transport)
	
	return &MCPClient{
		client:    client,
		transport: transport,
	}, nil
}

// Initialize initializes the MCP client
func (c *MCPClient) Initialize(ctx context.Context) error {
	// Initialize the MCP client
	_, err := c.client.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	
	// List available tools
	tools, err := c.client.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}
	
	// Store available tools
	c.availableTools = make([]ToolDefinition, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		desc := ""
		if tool.Description != nil {
			desc = *tool.Description
		}
		
		c.availableTools = append(c.availableTools, ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
		})
		
		log.Printf("Found MCP tool: %s. Description: %s", tool.Name, desc)
	}
	
	return nil
}

// ListTools returns all available tools
func (c *MCPClient) ListTools() []ToolDefinition {
	return c.availableTools
}

// CallTool calls an MCP tool with the given arguments
func (c *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Check if the client is initialized
	if c.client == nil {
		return nil, errors.New("MCP client not initialized")
	}
	
	// Call the tool
	response, err := c.client.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP tool %s: %w", toolName, err)
	}
	
	// Extract the response content
	if response == nil || len(response.Content) == 0 || response.Content[0].TextContent == nil {
		return nil, errors.New("empty response from MCP tool")
	}
	
	// Return the response text
	return response.Content[0].TextContent.Text, nil
}

// CreateMCPTool creates a tool from an MCP tool definition
func (c *MCPClient) CreateMCPTool(toolDef ToolDefinition) (*MCPTool, error) {
	// Create a tool from the MCP tool definition
	return &MCPTool{
		mcpClient:   c,
		name:        toolDef.Name,
		description: toolDef.Description,
		parameters:  toolDef.Parameters,
	}, nil
}

// MCPTool represents a tool that calls an MCP endpoint
type MCPTool struct {
	mcpClient   *MCPClient
	name        string
	description *string
	parameters  map[string]interface{}
}

// Name returns the name of the tool
func (t *MCPTool) Name() string {
	return t.name
}

// Description returns a description of the tool
func (t *MCPTool) Description() string {
	if t.description == nil {
		return ""
	}
	return *t.description
}

// Parameters returns a map of parameter names to their schema definitions
func (t *MCPTool) Parameters() interface{} {
	return t.parameters
}

// Execute executes the tool with the given arguments
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return t.mcpClient.CallTool(ctx, t.name, args)
}
