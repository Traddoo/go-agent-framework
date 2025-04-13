# Go Agent Framework

A powerful and flexible framework for building multi-agent systems in Go, designed for simplicity, composability, and real-world use cases.

## Overview

Go Agent Framework provides a complete toolkit for building multi-agent systems where LLMs act as agent brains. It implements the patterns described in [Anthropic's Building Effective Agents](https://www.anthropic.com/news/building-effective-agents) research and provides a unified environment for agent coordination, communication, and shared state management.

Key features:

- 🧠 **LLM Integration** - Seamless integration with Anthropic and OpenAI LLMs
- 🔧 **Tool Support** - Rich tool ecosystem with first-class MCP (Model Context Protocol) support
- 🤝 **Multi-Agent Coordination** - Built-in support for agent teams and collaboration
- 📝 **Shared Memory** - Document store for shared knowledge and state management
- 📡 **Communication** - Inter-agent messaging with pub/sub patterns
- 🔄 **Workflow Patterns** - Ready-to-use workflow implementations for common patterns
- 🔍 **Monitoring & Tracing** - Built-in observability for debugging and analysis

## Installation

```bash
go get github.com/thomasfaulds/go-agent-framework
```

## Architecture

The Go Agent Framework is organized around these core components:

### Runtime

The runtime is the central coordination layer that manages agent lifecycles, resources, and task scheduling. It provides a unified API for creating and managing agents, teams, and tasks.

### Agents

Agents are the core building blocks of the framework. Each agent has:

- An LLM "brain" (Anthropic's Claude or OpenAI's GPT models)
- Task queues for handling work
- Access to tools
- Memory for storing information
- Communication capabilities

### Teams

Teams allow multiple agents to work together on complex tasks. The framework supports:

- Shared team memory
- Inter-agent communication
- Task distribution and coordination
- Different team strategies (parallel, sequential, orchestrator-workers)

### Workflows

The framework implements the workflow patterns described in Anthropic's research:

- **Prompt Chaining** - Breaking tasks into sequential steps
- **Routing** - Classifying inputs and directing to specialized handlers
- **Parallelization** - Running multiple tasks simultaneously
- **Orchestrator-Workers** - Central coordination with specialized workers
- **Evaluator-Optimizer** - Iterative refinement with feedback

### Memory / Document Store

The shared memory system allows agents to store and retrieve information:

- Document storage with versioning
- Memory chunks for semantic retrieval
- Locking mechanisms for concurrent access
- Search capabilities

### MCP Integration

First-class support for the Model Context Protocol:

- MCP client implementation
- MCP server wrapper
- Tool discovery and registration
- Seamless integration with agents

### Communication Bus

Communication mechanisms for inter-agent coordination:

- Message passing
- Pub/sub patterns
- Request-response patterns
- Topic-based routing

### Monitoring and Tracing

Built-in observability tools:

- Metrics collection
- Event recording
- Distributed tracing
- Span-based performance analysis

## Quick Start

Here's a simple example to create and run an agent:

```go
package main

import (
	"context"
	"log"
	
	"github.com/thomasfaulds/go-agent-framework/pkg/agent"
	"github.com/thomasfaulds/go-agent-framework/pkg/runtime"
)

func main() {
	ctx := context.Background()
	
	// Create the runtime
	rt, err := runtime.New(&runtime.Config{
		MaxAgents: 5,
		LogLevel:  "info",
	})
	if err != nil {
		log.Fatalf("Failed to create runtime: %v", err)
	}
	
	// Start the runtime
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop(ctx)
	
	// Create an agent
	agentConfig := &agent.Config{
		Name:               "MyAgent",
		Description:        "A helpful assistant",
		DefaultSystemPrompt: "You are a helpful AI assistant.",
		LLMProvider:        "anthropic",
		LLMModel:           "claude-3-5-sonnet",
	}
	
	myAgent, err := rt.CreateAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	
	// Create a task
	task := &runtime.Task{
		Description: "Answer a question",
		Input: map[string]interface{}{
			"question": "What is a multi-agent system?",
		},
	}
	
	// Submit the task to the agent
	taskID, err := rt.SubmitTask(ctx, task, myAgent.ID)
	if err != nil {
		log.Fatalf("Failed to submit task: %v", err)
	}
	
	log.Printf("Task submitted with ID: %s", taskID)
}
```

## Creating Teams

Here's how to create and use a team of agents:

```go
// Create individual agents
agent1, _ := rt.CreateAgent(ctx, agent1Config)
agent2, _ := rt.CreateAgent(ctx, agent2Config)
agent3, _ := rt.CreateAgent(ctx, agent3Config)

// Create a team
teamConfig := &agent.TeamConfig{
	Name:        "ResearchTeam",
	Description: "A team for research tasks",
	AgentIDs:    []string{agent1.ID, agent2.ID, agent3.ID},
	TeamStrategy: "orchestrator",
	CoordinatorID: agent1.ID,
}

team, err := rt.CreateTeam(ctx, teamConfig)
if err != nil {
	log.Fatalf("Failed to create team: %v", err)
}

// Submit a task to the team
taskID, err := rt.SubmitTask(ctx, task, team.ID)
```

## Using Workflows

The framework provides builders for creating different workflow patterns:

```go
// Create a prompt chaining workflow
workflow := workflow.NewPromptChainingBuilder("Document Creation", "Create a document in steps")
	.AddLLMStep("Generate Outline", "Create an outline for a document about {{topic}}")
	.AddLLMStep("Write Draft", "Write a draft based on this outline: {{outline}}")
	.AddLLMStep("Edit Draft", "Edit and improve this draft: {{draft}}")
	.Build()

// Submit a task with the workflow
task := &runtime.Task{
	Description: "Create a document about AI",
	Input: map[string]interface{}{
		"topic": "Artificial Intelligence",
	},
	Workflow: workflow,
}

taskID, _ := rt.SubmitTask(ctx, task, agent.ID)
```

## Working with MCP

Create and use MCP servers for extended tool capabilities:

```go
// Create an MCP server
serverConfig := &mcp.ServerConfig{
	Command: "go",
	Args:    []string{"run", "./path/to/mcp/server.go"},
}

mcpServer, err := mcp.NewMCPServer(serverConfig)
if err != nil {
	log.Fatalf("Failed to create MCP server: %v", err)
}

// Start the server
if err := mcpServer.Start(ctx); err != nil {
	log.Fatalf("Failed to start MCP server: %v", err)
}
defer mcpServer.Stop()

// List available tools
tools, err := mcpServer.ListTools(ctx)
if err != nil {
	log.Fatalf("Failed to list tools: %v", err)
}

for _, tool := range tools {
	log.Printf("Found tool: %s - %s", tool.Name, tool.Description)
}

// Call a tool
result, err := mcpServer.CallTool(ctx, "hello", map[string]interface{}{
	"name": "World",
})
if err != nil {
	log.Fatalf("Failed to call tool: %v", err)
}

log.Printf("Tool result: %v", result)
```

## Shared Memory Example

Using the shared document store:

```go
// Create a memory store
store := memory.NewInMemoryStore()
store.Initialize(ctx)

// Create a document
doc := &memory.Document{
	Title:   "Research Notes",
	Content: "# Initial Research Notes\n\nThis document will contain our research findings.",
	Tags:    []string{"research", "notes"},
}

if err := store.CreateDocument(ctx, doc); err != nil {
	log.Fatalf("Failed to create document: %v", err)
}

// Update the document
doc.Content += "\n\n## New Finding\n\nWe discovered something interesting today."
if err := store.UpdateDocument(ctx, doc); err != nil {
	log.Fatalf("Failed to update document: %v", err)
}

// Search for documents
results, err := store.SearchDocuments(ctx, "research finding")
if err != nil {
	log.Fatalf("Failed to search documents: %v", err)
}

for _, result := range results {
	log.Printf("Found document: %s", result.Title)
}
```

## Communication Between Agents

Using the communication bus:

```go
// Create a communication bus
bus := comms.NewInMemoryBus()
bus.Initialize(ctx)

// Create a message
message := comms.NewMessage("agent1", "agent2", comms.MessageTypes.Command, "Please analyze this data")
message.SetHeader(comms.Headers.ContentType, comms.ContentTypes.Text)

// Send the message
if err := bus.Publish(ctx, "agent2", message); err != nil {
	log.Fatalf("Failed to publish message: %v", err)
}

// Subscribe to messages
subscriptionID, err := bus.Subscribe(ctx, "agent2", func(ctx context.Context, msg *comms.Message) error {
	log.Printf("Received message from %s: %v", msg.From, msg.Content)
	return nil
})

if err != nil {
	log.Fatalf("Failed to subscribe: %v", err)
}

// Unsubscribe when done
defer bus.Unsubscribe(ctx, subscriptionID)
```

## Advanced Configuration

The framework offers extensive configuration options:

```go
// Create a runtime with advanced configuration
rt, err := runtime.New(&runtime.Config{
	MaxAgents:         20,
	DefaultLLMConfig:  map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  4096,
	},
	ResourceLimits: runtime.ResourceLimits{
		MaxConcurrentLLMCalls: 10,
		MaxMemoryUsagePerAgent: 1 << 20, // 1MB
		TokenBudgetPerMinute:   5000,
	},
	LogLevel:        "debug",
	Persistence:     true,
	PersistencePath: "./data",
})
```

## Monitoring & Observability

Using the built-in monitoring system:

```go
// Create a monitoring system
monitor := monitoring.NewInMemorySystem()
monitor.Initialize(ctx)

// Record metrics
monitor.RecordMetric(ctx, monitoring.MetricNames.LLMCalls, 1, map[string]string{
	"model": "claude-3-5-sonnet",
	"agent": "agent1",
})

// Record events
monitor.RecordEvent(ctx, monitoring.EventNames.TaskStarted, map[string]interface{}{
	"task_id": "task-123",
	"agent_id": "agent1",
})

// Create spans for timing
ctx, span := monitor.StartSpan(ctx, "process_task")
// Do some work...
span.AddTag("task_type", "research")
span.End()
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
