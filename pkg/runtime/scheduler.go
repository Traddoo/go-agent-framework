package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Scheduler manages task execution across agents
type Scheduler struct {
	runtime    *Runtime
	tasks      map[string]*Task
	statuses   map[string]*TaskStatus
	taskQueue  []*Task
	mutex      sync.RWMutex
	
	// Channels for task management
	taskCh     chan *Task
	resultCh   chan *TaskResult
	stopCh     chan struct{}
}

// TaskResult represents the result of a completed task
type TaskResult struct {
	TaskID     string
	Success    bool
	Result     map[string]interface{}
	Error      string
	FinishedAt time.Time
}

// NewScheduler creates a new scheduler
func NewScheduler(runtime *Runtime) *Scheduler {
	return &Scheduler{
		runtime:    runtime,
		tasks:      make(map[string]*Task),
		statuses:   make(map[string]*TaskStatus),
		taskQueue:  make([]*Task, 0),
		taskCh:     make(chan *Task, 100),
		resultCh:   make(chan *TaskResult, 100),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the scheduler's worker goroutines
func (s *Scheduler) Start(ctx context.Context) error {
	// Start worker goroutines to process tasks
	go s.processTaskQueue(ctx)
	go s.processTaskResults(ctx)
	
	return nil
}

// Stop stops the scheduler's worker goroutines
func (s *Scheduler) Stop(ctx context.Context) error {
	close(s.stopCh)
	return nil
}

// ScheduleTask adds a task to the scheduler's queue
func (s *Scheduler) ScheduleTask(ctx context.Context, task *Task) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if a task with this ID already exists
	if _, exists := s.tasks[task.ID]; exists {
		return errors.New("task with this ID already exists")
	}
	
	// Create a status for the task
	status := &TaskStatus{
		ID:         task.ID,
		State:      "pending",
		Progress:   0.0,
	}
	
	// Add the task and status to the scheduler
	s.tasks[task.ID] = task
	s.statuses[task.ID] = status
	s.taskQueue = append(s.taskQueue, task)
	
	// Send the task to the processing channel
	select {
	case s.taskCh <- task:
		// Task sent successfully
	default:
		// Channel is full, task will be processed from the queue
	}
	
	return nil
}

// GetTaskStatus retrieves the status of a task
func (s *Scheduler) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Check if the task exists
	status, exists := s.statuses[taskID]
	if !exists {
		return nil, errors.New("task not found")
	}
	
	return status, nil
}

// UpdateTaskStatus updates the status of a task
func (s *Scheduler) UpdateTaskStatus(ctx context.Context, taskID string, newStatus *TaskStatus) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if the task exists
	_, exists := s.statuses[taskID]
	if !exists {
		return errors.New("task not found")
	}
	
	// Update the status
	s.statuses[taskID] = newStatus
	
	return nil
}

// processTaskQueue processes tasks from the queue
func (s *Scheduler) processTaskQueue(ctx context.Context) {
	for {
		select {
		case <-s.stopCh:
			return
		case task := <-s.taskCh:
			// Process the task
			go s.executeTask(ctx, task)
		case <-time.After(100 * time.Millisecond):
			// Check if there are any tasks in the queue
			s.mutex.Lock()
			if len(s.taskQueue) > 0 {
				task := s.taskQueue[0]
				s.taskQueue = s.taskQueue[1:]
				go s.executeTask(ctx, task)
			}
			s.mutex.Unlock()
		}
	}
}

// processTaskResults processes task results
func (s *Scheduler) processTaskResults(ctx context.Context) {
	for {
		select {
		case <-s.stopCh:
			return
		case result := <-s.resultCh:
			// Update the task status
			s.mutex.Lock()
			status, exists := s.statuses[result.TaskID]
			if exists {
				status.CompletedAt = result.FinishedAt
				
				if result.Success {
					// For workflow tasks or regular tasks, mark as completed
					status.State = "completed"
					status.Progress = 1.0
					status.Result = result.Result
				} else {
					status.State = "failed"
					status.Error = result.Error
				}
			}
			s.mutex.Unlock()
		}
	}
}

// executeTask executes a task on the assigned agent or team
func (s *Scheduler) executeTask(ctx context.Context, task *Task) {
	// Update task status to running
	s.mutex.Lock()
	status := s.statuses[task.ID]
	status.State = "running"
	status.StartedAt = time.Now()
	s.mutex.Unlock()
	
	// Check if this is a workflow task
	if task.Workflow != nil && s.runtime.WorkflowEng != nil {
		// Create a dummy task result for now
		s.resultCh <- &TaskResult{
			TaskID:     task.ID,
			Success:    true,
			Result: map[string]interface{}{
				"message": "Workflow execution simulated (not fully implemented)",
			},
			FinishedAt: time.Now(),
		}
		return
	}
	
	// Determine if the task is assigned to an agent or a team
	var result *TaskResult
	
	if task.AssignedTo != "" {
		// Check if it's an agent
		agent, err := s.runtime.AgentMgr.GetAgent(ctx, task.AssignedTo)
		if err == nil {
			// Execute the task on the agent
			taskOutput, err := agent.ExecuteTask(ctx, task)
			
			if err != nil {
				result = &TaskResult{
					TaskID:     task.ID,
					Success:    false,
					Error:      err.Error(),
					FinishedAt: time.Now(),
				}
			} else {
				result = &TaskResult{
					TaskID:     task.ID,
					Success:    true,
					Result:     taskOutput,
					FinishedAt: time.Now(),
				}
			}
		} else {
			// Check if it's a team
			team, err := s.runtime.AgentMgr.GetTeam(ctx, task.AssignedTo)
			if err == nil {
				// Execute the task on the team
				taskOutput, err := team.ExecuteTask(ctx, task)
				
				if err != nil {
					result = &TaskResult{
						TaskID:     task.ID,
						Success:    false,
						Error:      err.Error(),
						FinishedAt: time.Now(),
					}
				} else {
					result = &TaskResult{
						TaskID:     task.ID,
						Success:    true,
						Result:     taskOutput,
						FinishedAt: time.Now(),
					}
				}
			} else {
				// Neither agent nor team found
				result = &TaskResult{
					TaskID:     task.ID,
					Success:    false,
					Error:      "assigned entity not found",
					FinishedAt: time.Now(),
				}
			}
		}
	} else {
		// No assigned entity
		result = &TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      "no agent or team assigned",
			FinishedAt: time.Now(),
		}
	}
	
	// Send the result back to the result channel
	s.resultCh <- result
}
