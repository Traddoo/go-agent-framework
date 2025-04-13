package agent

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Team represents a group of agents working together
type Team struct {
	ID          string
	Name        string
	Description string
	Agents      map[string]*Agent
	Coordinator *Agent       // Optional coordinator agent
	Config      *TeamConfig
	
	mutex       sync.RWMutex
}

// TeamConfig holds team configuration
type TeamConfig struct {
	ID           string
	Name         string
	Description  string
	AgentIDs     []string
	CoordinatorID string
	TeamStrategy string       // "parallel", "sequential", "orchestrator"
}

// NewTeam creates a new team with the given configuration
func NewTeam(cfg *TeamConfig) (*Team, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	
	if cfg.Name == "" {
		cfg.Name = "Team-" + cfg.ID[:8]
	}
	
	// Create a new team
	team := &Team{
		ID:          cfg.ID,
		Name:        cfg.Name,
		Description: cfg.Description,
		Agents:      make(map[string]*Agent),
		Config:      cfg,
	}
	
	return team, nil
}

// AddAgent adds an agent to the team
func (t *Team) AddAgent(agent *Agent) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// Check if the agent is already in the team
	if _, exists := t.Agents[agent.ID]; exists {
		return errors.New("agent already in team")
	}
	
	// Add the agent to the team
	t.Agents[agent.ID] = agent
	
	// Check if this agent should be the coordinator
	if t.Config.CoordinatorID == agent.ID {
		t.Coordinator = agent
	}
	
	return nil
}

// RemoveAgent removes an agent from the team
func (t *Team) RemoveAgent(agentID string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// Check if the agent is in the team
	if _, exists := t.Agents[agentID]; !exists {
		return errors.New("agent not found in team")
	}
	
	// Remove the agent from the team
	delete(t.Agents, agentID)
	
	// Reset the coordinator if it was this agent
	if t.Coordinator != nil && t.Coordinator.ID == agentID {
		t.Coordinator = nil
	}
	
	return nil
}

// ExecuteTask executes a task using the team's strategy
func (t *Team) ExecuteTask(ctx context.Context, task interface{}) (map[string]interface{}, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	// Check if we have agents in the team
	if len(t.Agents) == 0 {
		return nil, errors.New("no agents in team")
	}
	
	// Choose a strategy based on the team's configuration
	switch t.Config.TeamStrategy {
	case "parallel":
		return t.executeParallel(ctx, task)
	case "sequential":
		return t.executeSequential(ctx, task)
	case "orchestrator":
		return t.executeOrchestrator(ctx, task)
	default:
		// Default to parallel execution
		return t.executeParallel(ctx, task)
	}
}

// executeParallel executes a task in parallel across all agents
func (t *Team) executeParallel(ctx context.Context, task interface{}) (map[string]interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	// Create channels for results and errors
	resultCh := make(chan map[string]interface{}, len(t.Agents))
	errCh := make(chan error, len(t.Agents))
	
	// Launch a goroutine for each agent
	for _, agent := range t.Agents {
		go func(a *Agent) {
			result, err := a.ExecuteTask(ctx, task)
			if err != nil {
				errCh <- err
			} else {
				resultCh <- result
			}
		}(agent)
	}
	
	// Collect results
	results := make(map[string]interface{})
	errors := make([]string, 0)
	
	// Wait for all agents to complete or context to cancel
	for i := 0; i < len(t.Agents); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			errors = append(errors, err.Error())
		case result := <-resultCh:
			for k, v := range result {
				// Prefix the key with the agent's ID to avoid collisions
				results[k] = v
			}
		}
	}
	
	// Add errors to results
	if len(errors) > 0 {
		results["errors"] = errors
	}
	
	return results, nil
}

// executeSequential executes a task sequentially across agents
func (t *Team) executeSequential(ctx context.Context, task interface{}) (map[string]interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	
	// Convert task to a map if it's not already
	taskMap, ok := task.(map[string]interface{})
	if !ok {
		taskMap = map[string]interface{}{
			"task": task,
		}
	}
	
	// Execute the task on each agent sequentially, passing the results to the next
	currentTask := taskMap
	results := make(map[string]interface{})
	
	for _, agent := range t.Agents {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		
		// Execute the task on this agent
		result, err := agent.ExecuteTask(ctx, currentTask)
		if err != nil {
			return nil, err
		}
		
		// Update results
		for k, v := range result {
			results[k] = v
		}
		
		// Update the task for the next agent
		currentTask = result
	}
	
	return results, nil
}

// executeOrchestrator executes a task using an orchestrator-workers pattern
func (t *Team) executeOrchestrator(ctx context.Context, task interface{}) (map[string]interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	
	// Check if we have a coordinator
	if t.Coordinator == nil {
		return nil, errors.New("no coordinator agent assigned")
	}
	
	// Convert task to a map if it's not already
	taskMap, ok := task.(map[string]interface{})
	if !ok {
		taskMap = map[string]interface{}{
			"task": task,
		}
	}
	
	// Add worker agents to the task
	workers := make(map[string]interface{})
	for id, agent := range t.Agents {
		if id != t.Coordinator.ID {
			workers[id] = map[string]interface{}{
				"id":   agent.ID,
				"name": agent.Name,
			}
		}
	}
	
	taskMap["workers"] = workers
	
	// Execute the task on the coordinator
	initialPlan, err := t.Coordinator.ExecuteTask(ctx, taskMap)
	if err != nil {
		return nil, err
	}
	
	// The coordinator should return a plan with subtasks for workers
	plan, ok := initialPlan["plan"].(map[string]interface{})
	if !ok {
		return nil, errors.New("coordinator did not return a valid plan")
	}
	
	// Execute subtasks on workers
	subtasks, ok := plan["subtasks"].(map[string]interface{})
	if !ok {
		return nil, errors.New("no subtasks in plan")
	}
	
	// Create channels for results and errors
	resultCh := make(chan map[string]interface{}, len(subtasks))
	errCh := make(chan error, len(subtasks))
	
	// Launch goroutines for each subtask
	for workerID, subtask := range subtasks {
		// Find the worker agent
		worker, exists := t.Agents[workerID]
		if !exists {
			continue // Skip if worker not found
		}
		
		go func(a *Agent, st interface{}) {
			result, err := a.ExecuteTask(ctx, st)
			if err != nil {
				errCh <- err
			} else {
				resultCh <- result
			}
		}(worker, subtask)
	}
	
	// Collect results
	workerResults := make(map[string]interface{})
	errors := make([]string, 0)
	
	// Wait for all workers to complete or context to cancel
	for i := 0; i < len(subtasks); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			errors = append(errors, err.Error())
		case result := <-resultCh:
			for k, v := range result {
				workerResults[k] = v
			}
		}
	}
	
	// Add errors to results
	if len(errors) > 0 {
		workerResults["errors"] = errors
	}
	
	// Send the results back to the coordinator for final synthesis
	finalTaskMap := map[string]interface{}{
		"task":          taskMap["task"],
		"workerResults": workerResults,
		"action":        "synthesize",
	}
	
	finalResult, err := t.Coordinator.ExecuteTask(ctx, finalTaskMap)
	if err != nil {
		return nil, err
	}
	
	return finalResult, nil
}
