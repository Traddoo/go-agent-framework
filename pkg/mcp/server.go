package mcp

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/traddoo/go-agent-framework/pkg/tools"
)

// MCPServer represents an MCP server that can be used by agents
type MCPServer struct {
	cmd         *exec.Cmd
	transport   transport.Transport
	serverTools []tools.Tool
	client      *mcp_golang.Client
}

// ServerConfig holds configuration for an MCP server
type ServerConfig struct {
	Command     string
	Args        []string
	ToolsToExpose []tools.Tool
}

// NewMCPServer creates a new MCP server
func NewMCPServer(cfg *ServerConfig) (*MCPServer, error) {
	// Create a command to run the server
	cmd := exec.Command(cfg.Command, cfg.Args...)
	
	// Get stdin and stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	
	// Create transport
	transport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	
	// Create MCP server
	server := &MCPServer{
		cmd:         cmd,
		transport:   transport,
		serverTools: cfg.ToolsToExpose,
	}
	
	return server, nil
}

// Start starts the MCP server
func (s *MCPServer) Start(ctx context.Context) error {
	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	// Create MCP client
	s.client = mcp_golang.NewClient(s.transport)
	
	// Initialize the client
	_, err := s.client.Initialize(ctx)
	if err != nil {
		// Kill the process on error
		s.cmd.Process.Kill()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	
	// Register tools
	for _, tool := range s.serverTools {
		// In a real implementation, we'd register the tool with the MCP server
		// For now, we'll just log that we're registering it
		log.Printf("Registering tool %s with MCP server", tool.Name())
	}
	
	return nil
}

// Stop stops the MCP server
func (s *MCPServer) Stop() error {
	// Kill the process
	if s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

// GetTools returns the tools exposed by the server
func (s *MCPServer) GetTools() []tools.Tool {
	return s.serverTools
}

// ListTools lists all available tools from the MCP server
func (s *MCPServer) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	// Check if client is initialized
	if s.client == nil {
		return nil, fmt.Errorf("MCP server not started")
	}
	
	// List tools
	tools, err := s.client.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}
	
	// Convert tools to our format
	toolDefs := make([]ToolDefinition, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolDefs = append(toolDefs, ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	
	return toolDefs, nil
}

// CallTool calls a tool on the MCP server
func (s *MCPServer) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Check if client is initialized
	if s.client == nil {
		return nil, fmt.Errorf("MCP server not started")
	}
	
	// Call the tool
	response, err := s.client.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP tool %s: %w", toolName, err)
	}
	
	// Extract the response content
	if response == nil || len(response.Content) == 0 || response.Content[0].TextContent == nil {
		return nil, fmt.Errorf("empty response from MCP tool")
	}
	
	// Return the response text
	return response.Content[0].TextContent.Text, nil
}
