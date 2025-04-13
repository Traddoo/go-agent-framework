package monitoring

import (
	"context"
	"sync"
	"time"
)

// System defines the monitoring system interface
type System interface {
	// Initialize initializes the monitoring system
	Initialize(ctx context.Context) error
	
	// RecordMetric records a metric
	RecordMetric(ctx context.Context, name string, value float64, tags map[string]string)
	
	// RecordEvent records an event
	RecordEvent(ctx context.Context, name string, properties map[string]interface{})
	
	// StartSpan starts a new trace span
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a trace span
type Span interface {
	// End ends the span
	End()
	
	// AddTag adds a tag to the span
	AddTag(key string, value string)
	
	// AddEvent adds an event to the span
	AddEvent(name string, attributes map[string]interface{})
}

// InMemorySystem implements the System interface with in-memory storage
type InMemorySystem struct {
	metrics map[string][]MetricPoint
	events  []Event
	mutex   sync.RWMutex
}

// MetricPoint represents a metric value at a point in time
type MetricPoint struct {
	Name      string
	Value     float64
	Tags      map[string]string
	Timestamp time.Time
}

// Event represents an event
type Event struct {
	Name       string
	Properties map[string]interface{}
	Timestamp  time.Time
}

// InMemorySpan represents a span in the in-memory system
type InMemorySpan struct {
	Name       string
	Tags       map[string]string
	Events     []SpanEvent
	StartTime  time.Time
	EndTime    time.Time
	System     *InMemorySystem
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string
	Attributes map[string]interface{}
	Timestamp  time.Time
}

// NewInMemorySystem creates a new in-memory monitoring system
func NewInMemorySystem() *InMemorySystem {
	return &InMemorySystem{
		metrics: make(map[string][]MetricPoint),
		events:  make([]Event, 0),
	}
}

// Initialize initializes the monitoring system
func (m *InMemorySystem) Initialize(ctx context.Context) error {
	// Nothing to initialize for in-memory system
	return nil
}

// RecordMetric records a metric
func (m *InMemorySystem) RecordMetric(ctx context.Context, name string, value float64, tags map[string]string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Create a metric point
	point := MetricPoint{
		Name:      name,
		Value:     value,
		Tags:      tags,
		Timestamp: time.Now(),
	}
	
	// Add the metric point to the list
	m.metrics[name] = append(m.metrics[name], point)
}

// RecordEvent records an event
func (m *InMemorySystem) RecordEvent(ctx context.Context, name string, properties map[string]interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Create an event
	event := Event{
		Name:       name,
		Properties: properties,
		Timestamp:  time.Now(),
	}
	
	// Add the event to the list
	m.events = append(m.events, event)
}

// StartSpan starts a new trace span
func (m *InMemorySystem) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	// Create a new span
	span := &InMemorySpan{
		Name:      name,
		Tags:      make(map[string]string),
		Events:    make([]SpanEvent, 0),
		StartTime: time.Now(),
		System:    m,
	}
	
	// Create a new context with the span
	newCtx := context.WithValue(ctx, spanContextKey, span)
	
	return newCtx, span
}

// End ends the span
func (s *InMemorySpan) End() {
	s.EndTime = time.Now()
	
	// Record the span duration as a metric
	duration := float64(s.EndTime.Sub(s.StartTime).Milliseconds())
	s.System.RecordMetric(context.Background(), "span_duration_ms", duration, map[string]string{
		"name": s.Name,
	})
}

// AddTag adds a tag to the span
func (s *InMemorySpan) AddTag(key string, value string) {
	s.Tags[key] = value
}

// AddEvent adds an event to the span
func (s *InMemorySpan) AddEvent(name string, attributes map[string]interface{}) {
	// Create a new event
	event := SpanEvent{
		Name:       name,
		Attributes: attributes,
		Timestamp:  time.Now(),
	}
	
	// Add the event to the span
	s.Events = append(s.Events, event)
}

// GetSpanFromContext gets a span from the context
func GetSpanFromContext(ctx context.Context) (Span, bool) {
	span, ok := ctx.Value(spanContextKey).(Span)
	return span, ok
}

// spanContextKey is the key used to store spans in the context
type spanContextKeyType struct{}
var spanContextKey = spanContextKeyType{}

// Common metric names
var MetricNames = struct {
	LLMCalls             string
	LLMTokens            string
	LLMLatency           string
	ToolCalls            string
	ToolLatency          string
	AgentTasks           string
	AgentTaskDuration    string
	WorkflowExecutions   string
	WorkflowStepDuration string
	MemoryOperations     string
	MemoryLatency        string
}{
	LLMCalls:             "llm.calls",
	LLMTokens:            "llm.tokens",
	LLMLatency:           "llm.latency_ms",
	ToolCalls:            "tool.calls",
	ToolLatency:          "tool.latency_ms",
	AgentTasks:           "agent.tasks",
	AgentTaskDuration:    "agent.task_duration_ms",
	WorkflowExecutions:   "workflow.executions",
	WorkflowStepDuration: "workflow.step_duration_ms",
	MemoryOperations:     "memory.operations",
	MemoryLatency:        "memory.latency_ms",
}

// Common event names
var EventNames = struct {
	AgentCreated     string
	AgentTerminated  string
	TaskStarted      string
	TaskCompleted    string
	TaskFailed       string
	WorkflowStarted  string
	WorkflowCompleted string
	WorkflowFailed   string
	LLMError         string
	ToolError        string
}{
	AgentCreated:     "agent.created",
	AgentTerminated:  "agent.terminated",
	TaskStarted:      "task.started",
	TaskCompleted:    "task.completed",
	TaskFailed:       "task.failed",
	WorkflowStarted:  "workflow.started",
	WorkflowCompleted: "workflow.completed",
	WorkflowFailed:   "workflow.failed",
	LLMError:         "llm.error",
	ToolError:        "tool.error",
}
