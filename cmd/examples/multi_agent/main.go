package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/traddoo/go-agent-framework/pkg/agent"
	"github.com/traddoo/go-agent-framework/pkg/comms"
	"github.com/traddoo/go-agent-framework/pkg/memory"
	"github.com/traddoo/go-agent-framework/pkg/runtime"
	"github.com/traddoo/go-agent-framework/pkg/tools"
	"github.com/traddoo/go-agent-framework/pkg/workflow"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create the runtime
	rt, err := runtime.New(&runtime.Config{
		MaxAgents:       10,
		DefaultLLMConfig: map[string]interface{}{"temperature": 0.7},
		ResourceLimits: runtime.ResourceLimits{
			MaxConcurrentLLMCalls: 5,
			MaxMemoryUsagePerAgent: 1 << 20, // 1MB
		},
		LogLevel: "info",
	})
	if err != nil {
		log.Fatalf("Failed to create runtime: %v", err)
	}

	// Start the runtime
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop(ctx)

	// Create a shared memory store
	sharedMemory := memory.NewInMemoryStore()
	sharedMemory.Initialize(ctx)

	// Create a communication bus for agent coordination
	commBus := comms.NewInMemoryBus()
	commBus.Initialize(ctx)

	// Register standard tools
	toolRegistry := tools.NewToolRegistry()
	registerStandardTools(toolRegistry, sharedMemory, commBus)

	// Create a document in shared memory for agents to collaborate on
	projectDoc := &memory.Document{
		ID:        "project-123",
		Title:     "AI Research Project",
		Content:   "# AI Research Project\n\nThis is a collaborative research project on multi-agent systems.",
		Tags:      []string{"research", "ai", "multi-agent"},
		CreatedBy: "system",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"status": "in-progress",
		},
	}
	
	if err := sharedMemory.CreateDocument(ctx, projectDoc); err != nil {
		log.Fatalf("Failed to create document: %v", err)
	}

	// Create the research agent
	researchAgentConfig := &agent.Config{
		Name:               "ResearchAgent",
		Description:        "An agent specialized in research and information gathering",
		MaxConcurrentTasks: 1,
		DefaultSystemPrompt: `You are ResearchAgent, an AI specialized in research and information gathering.
Your role is to collect, analyze, and summarize information on various topics.
You have access to tools to search for information and read documents.`,
		LLMProvider:     "anthropic",
		LLMModel:        "claude-3-5-sonnet",
		ModelParameters: map[string]interface{}{"temperature": 0.2},
	}
	
	researchAgent, err := rt.CreateAgent(ctx, researchAgentConfig)
	if err != nil {
		log.Fatalf("Failed to create research agent: %v", err)
	}
	
	// Register tools with research agent
	for _, tool := range toolRegistry.ListTools() {
		if err := researchAgent.RegisterTool(tool); err != nil {
			log.Printf("Warning: Failed to register tool %s with research agent: %v", tool.Name(), err)
		}
	}

	// Create the writing agent
	writingAgentConfig := &agent.Config{
		Name:               "WritingAgent",
		Description:        "An agent specialized in writing and content creation",
		MaxConcurrentTasks: 1,
		DefaultSystemPrompt: `You are WritingAgent, an AI specialized in writing and content creation.
Your role is to create high-quality, well-structured documents based on information provided.
You have access to tools to read and update documents.`,
		LLMProvider:     "anthropic",
		LLMModel:        "claude-3-5-sonnet",
		ModelParameters: map[string]interface{}{"temperature": 0.7},
	}
	
	writingAgent, err := rt.CreateAgent(ctx, writingAgentConfig)
	if err != nil {
		log.Fatalf("Failed to create writing agent: %v", err)
	}
	
	// Register tools with writing agent
	for _, tool := range toolRegistry.ListTools() {
		if err := writingAgent.RegisterTool(tool); err != nil {
			log.Printf("Warning: Failed to register tool %s with writing agent: %v", tool.Name(), err)
		}
	}

	// Create the review agent
	reviewAgentConfig := &agent.Config{
		Name:               "ReviewAgent",
		Description:        "An agent specialized in reviewing and providing feedback",
		MaxConcurrentTasks: 1,
		DefaultSystemPrompt: `You are ReviewAgent, an AI specialized in reviewing and providing feedback.
Your role is to review content, check for accuracy, clarity, and completeness.
You have access to tools to read documents and provide feedback.`,
		LLMProvider:     "anthropic",
		LLMModel:        "claude-3-5-sonnet",
		ModelParameters: map[string]interface{}{"temperature": 0.3},
	}
	
	reviewAgent, err := rt.CreateAgent(ctx, reviewAgentConfig)
	if err != nil {
		log.Fatalf("Failed to create review agent: %v", err)
	}
	
	// Register tools with review agent
	for _, tool := range toolRegistry.ListTools() {
		if err := reviewAgent.RegisterTool(tool); err != nil {
			log.Printf("Warning: Failed to register tool %s with review agent: %v", tool.Name(), err)
		}
	}

	// Create a team with these agents
	teamConfig := &agent.TeamConfig{
		Name:        "ResearchTeam",
		Description: "A team of agents working together on research projects",
		AgentIDs:    []string{researchAgent.ID, writingAgent.ID, reviewAgent.ID},
		TeamStrategy: "orchestrator",
		CoordinatorID: researchAgent.ID, // Research agent coordinates the work
	}
	
	team, err := rt.CreateTeam(ctx, teamConfig)
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}

	// Create a workflow for the team
	workflow := createResearchWorkflow(researchAgent.ID, writingAgent.ID, reviewAgent.ID)

	// Create a task for the team with the workflow
	task := &runtime.Task{
		Description: "Research and write a comprehensive article on multi-agent systems",
		Input: map[string]interface{}{
			"topic": "Multi-Agent Systems in Artificial Intelligence",
			"requirements": map[string]interface{}{
				"sections": []string{
					"Introduction to Multi-Agent Systems",
					"Architecture and Components",
					"Communication Protocols",
					"Coordination Mechanisms",
					"Real-world Applications",
					"Future Directions",
				},
				"minLength": 2000,
				"maxLength": 5000,
				"format": "markdown",
				"style": "technical but accessible",
			},
			"documentID": projectDoc.ID,
			"deadline": time.Now().Add(30 * time.Minute),
		},
		Workflow: workflow,
		Priority: 1,
		Deadline: time.Now().Add(30 * time.Minute),
	}

	// Submit the task and wait for completion
	taskID, err := rt.SubmitTask(ctx, task, team.ID)
	if err != nil {
		log.Fatalf("Failed to submit task: %v", err)
	}

	// Poll for task completion
	for {
		status, err := rt.GetTaskStatus(ctx, taskID)
		if err != nil {
			log.Fatalf("Failed to get task status: %v", err)
		}

		log.Printf("Task status: %s, Progress: %.2f", status.State, status.Progress)

		if status.State == "completed" || status.State == "failed" {
			if status.State == "completed" {
				log.Printf("Task completed successfully")
				
				// Read the final document
				finalDoc, err := sharedMemory.ReadDocument(ctx, projectDoc.ID)
				if err != nil {
					log.Fatalf("Failed to read final document: %v", err)
				}
				
				// Create output directory if it doesn't exist
				outputDir := "output"
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					log.Fatalf("Failed to create output directory: %v", err)
				}
				
				// Save and display the final document
				outputPath := filepath.Join(outputDir, "research_project.md")
				if err := os.WriteFile(outputPath, []byte(finalDoc.Content), 0644); err != nil {
					log.Fatalf("Failed to save document: %v", err)
				}
				
				log.Printf("Final document saved to: %s", outputPath)
				log.Printf("Final document content:\n%s", finalDoc.Content)
			} else {
				log.Printf("Task failed: %v", status.Error)
			}
			break
		}

		time.Sleep(5 * time.Second)
	}
}

// createResearchWorkflow creates a workflow for the research team
func createResearchWorkflow(researchAgentID, writingAgentID, reviewAgentID string) *workflow.Workflow {
	// Create an orchestrator-workers workflow
	builder := workflow.NewOrchestratorBuilder(
		"Research Project Workflow",
		"A workflow for collaborative research and document creation",
	)
	
	// Set the research agent as the orchestrator
	builder.SetOrchestratorAgent("Research Coordinator", researchAgentID)
	
	// Add worker agents with their roles
	builder.AddWorkerAgent("Research Phase", researchAgentID)
	builder.AddWorkerAgent("Writing Phase", writingAgentID)
	builder.AddWorkerAgent("Review Phase", reviewAgentID)
	
	return builder.Build()
}

// registerStandardTools registers standard tools with the registry
func registerStandardTools(registry *tools.ToolRegistry, store memory.Interface, bus comms.Bus) {
	// Register document tools
	docReadTool := &DocumentReadTool{
		MemoryStore: store,
	}
	registry.RegisterTool(docReadTool)

	docUpdateTool := &DocumentUpdateTool{
		MemoryStore: store,
	}
	registry.RegisterTool(docUpdateTool)

	// Register information tools
	searchTool := &SearchTool{}
	registry.RegisterTool(searchTool)

	summarizeTool := &SummarizeTool{}
	registry.RegisterTool(summarizeTool)

	// Register communication tools
	messageTool := &MessageTool{
		CommBus: bus,
	}
	registry.RegisterTool(messageTool)
}

// DocumentReadTool reads a document from shared memory
type DocumentReadTool struct {
	MemoryStore memory.Interface
}

// Name returns the name of the tool
func (t *DocumentReadTool) Name() string {
	return "document_read"
}

// Description returns a description of the tool
func (t *DocumentReadTool) Description() string {
	return "Read a document from the shared memory"
}

// Parameters returns the parameters for the tool
func (t *DocumentReadTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"document_id": map[string]interface{}{
				"type": "string",
				"description": "The ID of the document to read",
			},
		},
		"required": []string{"document_id"},
	}
}

// Execute executes the tool
func (t *DocumentReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	documentID := args["document_id"].(string)
	
	// Read the document
	doc, err := t.MemoryStore.ReadDocument(ctx, documentID)
	if err != nil {
		return nil, tools.ToolError{
			Code:    "document_read_error",
			Message: "Failed to read document: " + err.Error(),
		}
	}
	
	return map[string]interface{}{
		"document": map[string]interface{}{
			"id":      doc.ID,
			"title":   doc.Title,
			"content": doc.Content,
			"tags":    doc.Tags,
			"metadata": doc.Metadata,
			"created_at": doc.CreatedAt,
			"updated_at": doc.UpdatedAt,
		},
	}, nil
}

// DocumentUpdateTool updates a document in shared memory
type DocumentUpdateTool struct {
	MemoryStore memory.Interface
}

// Name returns the name of the tool
func (t *DocumentUpdateTool) Name() string {
	return "document_update"
}

// Description returns a description of the tool
func (t *DocumentUpdateTool) Description() string {
	return "Update a document in the shared memory"
}

// Parameters returns the parameters for the tool
func (t *DocumentUpdateTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"document_id": map[string]interface{}{
				"type": "string",
				"description": "The ID of the document to update",
			},
			"content": map[string]interface{}{
				"type": "string",
				"description": "The new content of the document",
			},
			"title": map[string]interface{}{
				"type": "string",
				"description": "The new title of the document (optional)",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "The new tags for the document (optional)",
			},
			"metadata": map[string]interface{}{
				"type": "object",
				"description": "Additional metadata for the document (optional)",
			},
			"updated_by": map[string]interface{}{
				"type": "string",
				"description": "The ID of the agent updating the document",
			},
		},
		"required": []string{"document_id", "content", "updated_by"},
	}
}

// Execute executes the tool
func (t *DocumentUpdateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	documentID := args["document_id"].(string)
	content := args["content"].(string)
	updatedBy := args["updated_by"].(string)
	
	// Read the current document
	doc, err := t.MemoryStore.ReadDocument(ctx, documentID)
	if err != nil {
		return nil, tools.ToolError{
			Code:    "document_read_error",
			Message: "Failed to read document: " + err.Error(),
		}
	}
	
	// Update the document
	doc.Content = content
	doc.UpdatedBy = updatedBy
	doc.UpdatedAt = time.Now()
	
	// Update optional fields if provided
	if title, ok := args["title"].(string); ok {
		doc.Title = title
	}
	
	if tags, ok := args["tags"].([]interface{}); ok {
		newTags := make([]string, len(tags))
		for i, tag := range tags {
			newTags[i] = tag.(string)
		}
		doc.Tags = newTags
	}
	
	if metadata, ok := args["metadata"].(map[string]interface{}); ok {
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			doc.Metadata[k] = v
		}
	}
	
	// Save the updated document
	if err := t.MemoryStore.UpdateDocument(ctx, doc); err != nil {
		return nil, tools.ToolError{
			Code:    "document_update_error",
			Message: "Failed to update document: " + err.Error(),
		}
	}
	
	return map[string]interface{}{
		"success": true,
		"document_id": doc.ID,
		"updated_at": doc.UpdatedAt,
	}, nil
}

// SearchTool simulates searching for information
type SearchTool struct{}

// Name returns the name of the tool
func (t *SearchTool) Name() string {
	return "search"
}

// Description returns a description of the tool
func (t *SearchTool) Description() string {
	return "Search for information on a given topic"
}

// Parameters returns the parameters for the tool
func (t *SearchTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type": "string",
				"description": "The search query",
			},
			"max_results": map[string]interface{}{
				"type": "integer",
				"default": 5,
				"description": "Maximum number of results to return",
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool
func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	query := args["query"].(string)
	
	// Get max results parameter (default to 5)
	maxResults := 5
	if maxResultsArg, ok := args["max_results"].(float64); ok {
		maxResults = int(maxResultsArg)
	}
	
	// Simulate a search by generating relevant results based on the query
	// This is more dynamic than hardcoded results
	
	// First, generate basic results for any search about multi-agent systems
	basicResults := []map[string]interface{}{
		{
			"title": "Introduction to Multi-Agent Systems",
			"url": "https://example.com/multi-agent-systems",
			"snippet": "Multi-agent systems (MAS) are a field of study in artificial intelligence focused on systems composed of multiple interacting intelligent agents.",
		},
		{
			"title": "Agent Communication Languages",
			"url": "https://example.com/agent-communication",
			"snippet": "Agent Communication Languages (ACLs) provide agents with a means of exchanging information and knowledge.",
		},
		{
			"title": "Coordination in Multi-Agent Systems",
			"url": "https://example.com/coordination",
			"snippet": "Coordination is essential in multi-agent systems to ensure agents work together effectively toward common goals.",
		},
		{
			"title": "Architectures for Multi-Agent Systems",
			"url": "https://example.com/architectures",
			"snippet": "Different architectures for multi-agent systems include centralized, hierarchical, and distributed approaches, each with different tradeoffs.",
		},
		{
			"title": "Applications of Multi-Agent Systems",
			"url": "https://example.com/applications",
			"snippet": "Multi-agent systems are used in various domains including robotics, traffic management, e-commerce, and distributed problem solving.",
		},
	}
	
	// Topic-specific results for various query types
	topicResults := map[string][]map[string]interface{}{
		"architecture": {
			{
				"title": "BDI Architecture for Intelligent Agents",
				"url": "https://example.com/bdi-architecture",
				"snippet": "The Belief-Desire-Intention (BDI) architecture is a framework for modeling intelligent agents based on mental attitudes.",
			},
			{
				"title": "Layered Agent Architectures",
				"url": "https://example.com/layered-architectures",
				"snippet": "Layered architectures organize agent capabilities into hierarchical layers, from reactive behaviors to deliberative reasoning.",
			},
		},
		"communication": {
			{
				"title": "FIPA Agent Communication Language",
				"url": "https://example.com/fipa-acl",
				"snippet": "FIPA ACL is a standard language for agent communication based on speech act theory, defining message types like inform, request, and query.",
			},
			{
				"title": "Knowledge Query and Manipulation Language (KQML)",
				"url": "https://example.com/kqml",
				"snippet": "KQML is a language and protocol for exchanging information and knowledge between agents in a multi-agent system.",
			},
		},
		"coordination": {
			{
				"title": "Contract Net Protocol for Multi-Agent Coordination",
				"url": "https://example.com/contract-net",
				"snippet": "The Contract Net Protocol is a task allocation mechanism where agents bid for tasks based on their capabilities and resources.",
			},
			{
				"title": "Blackboard Systems for Agent Coordination",
				"url": "https://example.com/blackboard-systems",
				"snippet": "Blackboard systems provide a shared workspace where agents can post and access information, facilitating indirect coordination.",
			},
		},
		"application": {
			{
				"title": "Multi-Agent Systems in Disaster Response",
				"url": "https://example.com/disaster-response",
				"snippet": "Multi-agent systems are used in disaster response to coordinate heterogeneous teams of robots and human responders.",
			},
			{
				"title": "Trading Agent Competition",
				"url": "https://example.com/trading-agents",
				"snippet": "The Trading Agent Competition showcases multi-agent systems in e-commerce scenarios, with agents competing in simulated marketplaces.",
			},
		},
	}
	
	// Process the query to find relevant results
	results := make([]map[string]interface{}, 0)
	
	// First, add basic results
	results = append(results, basicResults...)
	
	// Then, check if query contains any of our special topics
	for topic, topicRes := range topicResults {
		if strings.Contains(strings.ToLower(query), strings.ToLower(topic)) {
			// Add the topic-specific results
			results = append(results, topicRes...)
		}
	}
	
	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	
	return map[string]interface{}{
		"results": results,
		"query": query,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// SummarizeTool simulates summarizing text
type SummarizeTool struct{}

// Name returns the name of the tool
func (t *SummarizeTool) Name() string {
	return "summarize"
}

// Description returns a description of the tool
func (t *SummarizeTool) Description() string {
	return "Summarize a piece of text"
}

// Parameters returns the parameters for the tool
func (t *SummarizeTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{
				"type": "string",
				"description": "The text to summarize",
			},
			"max_length": map[string]interface{}{
				"type": "integer",
				"default": 100,
				"description": "Maximum length of the summary in words",
			},
		},
		"required": []string{"text"},
	}
}

// Execute executes the tool
func (t *SummarizeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	text := args["text"].(string)
	
	// In a real implementation, this would use an LLM to summarize the text
	// For this example, we'll just return a mock summary
	return map[string]interface{}{
		"summary": "This is a summary of the provided text. In a real implementation, this would use an LLM to generate a proper summary.",
		"original_length": len(text),
	}, nil
}

// MessageTool sends messages between agents
type MessageTool struct {
	CommBus comms.Bus
}

// Name returns the name of the tool
func (t *MessageTool) Name() string {
	return "send_message"
}

// Description returns a description of the tool
func (t *MessageTool) Description() string {
	return "Send a message to another agent"
}

// Parameters returns the parameters for the tool
func (t *MessageTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"to": map[string]interface{}{
				"type": "string",
				"description": "The ID of the recipient agent",
			},
			"from": map[string]interface{}{
				"type": "string",
				"description": "The ID of the sender agent",
			},
			"content": map[string]interface{}{
				"type": "string",
				"description": "The message content",
			},
			"message_type": map[string]interface{}{
				"type": "string",
				"default": "text",
				"enum": []string{"text", "request", "response", "command"},
				"description": "The type of message",
			},
		},
		"required": []string{"to", "from", "content"},
	}
}

// Execute executes the tool
func (t *MessageTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	to := args["to"].(string)
	from := args["from"].(string)
	content := args["content"].(string)
	
	// Get message type (default to "text")
	messageType := "text"
	if msgType, ok := args["message_type"].(string); ok {
		messageType = msgType
	}
	
	// Create a message
	message := comms.NewMessage(from, to, messageType, content)
	
	// Send the message
	err := t.CommBus.Publish(ctx, to, message)
	if err != nil {
		return nil, tools.ToolError{
			Code:    "message_send_error",
			Message: "Failed to send message: " + err.Error(),
		}
	}
	
	return map[string]interface{}{
		"success": true,
		"message_id": message.ID,
		"timestamp": message.Timestamp,
	}, nil
}
