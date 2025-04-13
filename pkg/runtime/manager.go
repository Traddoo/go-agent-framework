package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/traddoo/go-agent-framework/pkg/agent"
)

// AgentManager handles agent lifecycle management
type AgentManager struct {
	runtime *Runtime
	agents  map[string]*agent.Agent
	teams   map[string]*agent.Team
	mutex   sync.RWMutex
}

// NewAgentManager creates a new agent manager
func NewAgentManager(runtime *Runtime) *AgentManager {
	return &AgentManager{
		runtime: runtime,
		agents:  make(map[string]*agent.Agent),
		teams:   make(map[string]*agent.Team),
	}
}

// RegisterAgent registers an agent with the manager
func (m *AgentManager) RegisterAgent(ctx context.Context, agent *agent.Agent) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if an agent with this ID already exists
	if _, exists := m.agents[agent.ID]; exists {
		return errors.New("agent with this ID already exists")
	}
	
	// Check if we've reached the maximum number of agents
	if len(m.agents) >= m.runtime.Config.MaxAgents {
		return errors.New("maximum number of agents reached")
	}
	
	// Add the agent to the registry
	m.agents[agent.ID] = agent
	
	return nil
}

// UnregisterAgent removes an agent from the manager
func (m *AgentManager) UnregisterAgent(ctx context.Context, agentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if the agent exists
	if _, exists := m.agents[agentID]; !exists {
		return errors.New("agent not found")
	}
	
	// Remove the agent from the registry
	delete(m.agents, agentID)
	
	return nil
}

// GetAgent retrieves an agent by ID
func (m *AgentManager) GetAgent(ctx context.Context, agentID string) (*agent.Agent, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Check if the agent exists
	agent, exists := m.agents[agentID]
	if !exists {
		return nil, errors.New("agent not found")
	}
	
	return agent, nil
}

// ListAgents lists all registered agents
func (m *AgentManager) ListAgents(ctx context.Context) ([]*agent.Agent, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Create a slice with all agents
	agents := make([]*agent.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// RegisterTeam registers a team with the manager
func (m *AgentManager) RegisterTeam(ctx context.Context, team *agent.Team) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if a team with this ID already exists
	if _, exists := m.teams[team.ID]; exists {
		return errors.New("team with this ID already exists")
	}
	
	// Add the team to the registry
	m.teams[team.ID] = team
	
	return nil
}

// UnregisterTeam removes a team from the manager
func (m *AgentManager) UnregisterTeam(ctx context.Context, teamID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if the team exists
	if _, exists := m.teams[teamID]; !exists {
		return errors.New("team not found")
	}
	
	// Remove the team from the registry
	delete(m.teams, teamID)
	
	return nil
}

// GetTeam retrieves a team by ID
func (m *AgentManager) GetTeam(ctx context.Context, teamID string) (*agent.Team, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Check if the team exists
	team, exists := m.teams[teamID]
	if !exists {
		return nil, errors.New("team not found")
	}
	
	return team, nil
}

// ListTeams lists all registered teams
func (m *AgentManager) ListTeams(ctx context.Context) ([]*agent.Team, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Create a slice with all teams
	teams := make([]*agent.Team, 0, len(m.teams))
	for _, team := range m.teams {
		teams = append(teams, team)
	}
	
	return teams, nil
}

// WaitForRunningTasks waits for all running tasks to complete
func (m *AgentManager) WaitForRunningTasks() {
	// This is a placeholder implementation
	// In a real implementation, you would check each agent and team
	// for running tasks and wait for them to complete
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// For now, we'll just count the number of agents and teams
	agentCount := len(m.agents)
	teamCount := len(m.teams)
	
	if agentCount+teamCount > 0 {
		// Give a bit of time for agents to complete their tasks
		time.Sleep(5 * time.Second)
	}
}
