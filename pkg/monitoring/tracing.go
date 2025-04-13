package monitoring

import (
	"context"
	"sync"
	"time"
)

// Tracer defines the tracer interface
type Tracer interface {
	// StartTrace starts a new trace
	StartTrace(ctx context.Context, name string) (context.Context, Trace)
	
	// GetTrace retrieves a trace by ID
	GetTrace(ctx context.Context, traceID string) (Trace, error)
	
	// ListTraces lists all traces
	ListTraces(ctx context.Context) []Trace
}

// Trace represents a trace
type Trace interface {
	// ID returns the trace ID
	ID() string
	
	// Name returns the trace name
	Name() string
	
	// StartTime returns the trace start time
	StartTime() time.Time
	
	// EndTime returns the trace end time
	EndTime() time.Time
	
	// AddSpan adds a span to the trace
	AddSpan(span TraceSpan)
	
	// End ends the trace
	End()
}

// TraceSpan represents a span within a trace
type TraceSpan interface {
	// ID returns the span ID
	ID() string
	
	// Name returns the span name
	Name() string
	
	// TraceID returns the parent trace ID
	TraceID() string
	
	// ParentID returns the parent span ID
	ParentID() string
	
	// StartTime returns the span start time
	StartTime() time.Time
	
	// EndTime returns the span end time
	EndTime() time.Time
	
	// AddEvent adds an event to the span
	AddEvent(name string, attributes map[string]interface{})
	
	// AddTag adds a tag to the span
	AddTag(key string, value string)
	
	// End ends the span
	End()
}

// InMemoryTracer implements the Tracer interface with in-memory storage
type InMemoryTracer struct {
	traces map[string]*InMemoryTrace
	mutex  sync.RWMutex
}

// InMemoryTrace implements the Trace interface
type InMemoryTrace struct {
	id        string
	name      string
	spans     map[string]*InMemoryTraceSpan
	startTime time.Time
	endTime   time.Time
	tracer    *InMemoryTracer
	mutex     sync.RWMutex
}

// InMemoryTraceSpan implements the TraceSpan interface
type InMemoryTraceSpan struct {
	id        string
	name      string
	traceID   string
	parentID  string
	startTime time.Time
	endTime   time.Time
	events    []TraceEvent
	tags      map[string]string
	trace     *InMemoryTrace
	mutex     sync.RWMutex
}

// TraceEvent represents an event within a span
type TraceEvent struct {
	Name       string
	Attributes map[string]interface{}
	Timestamp  time.Time
}

// NewInMemoryTracer creates a new in-memory tracer
func NewInMemoryTracer() *InMemoryTracer {
	return &InMemoryTracer{
		traces: make(map[string]*InMemoryTrace),
	}
}

// StartTrace starts a new trace
func (t *InMemoryTracer) StartTrace(ctx context.Context, name string) (context.Context, Trace) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// Create a new trace
	trace := &InMemoryTrace{
		id:        generateID(),
		name:      name,
		spans:     make(map[string]*InMemoryTraceSpan),
		startTime: time.Now(),
		tracer:    t,
	}
	
	// Add the trace to the map
	t.traces[trace.id] = trace
	
	// Create a new context with the trace
	newCtx := context.WithValue(ctx, traceContextKey, trace)
	
	return newCtx, trace
}

// GetTrace retrieves a trace by ID
func (t *InMemoryTracer) GetTrace(ctx context.Context, traceID string) (Trace, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	// Check if the trace exists
	trace, exists := t.traces[traceID]
	if !exists {
		return nil, ErrTraceNotFound
	}
	
	return trace, nil
}

// ListTraces lists all traces
func (t *InMemoryTracer) ListTraces(ctx context.Context) []Trace {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	// Create a slice with all traces
	traces := make([]Trace, 0, len(t.traces))
	for _, trace := range t.traces {
		traces = append(traces, trace)
	}
	
	return traces
}

// ID returns the trace ID
func (t *InMemoryTrace) ID() string {
	return t.id
}

// Name returns the trace name
func (t *InMemoryTrace) Name() string {
	return t.name
}

// StartTime returns the trace start time
func (t *InMemoryTrace) StartTime() time.Time {
	return t.startTime
}

// EndTime returns the trace end time
func (t *InMemoryTrace) EndTime() time.Time {
	return t.endTime
}

// AddSpan adds a span to the trace
func (t *InMemoryTrace) AddSpan(span TraceSpan) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// Add the span to the map
	inMemorySpan, ok := span.(*InMemoryTraceSpan)
	if ok {
		t.spans[inMemorySpan.id] = inMemorySpan
	}
}

// End ends the trace
func (t *InMemoryTrace) End() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.endTime = time.Now()
}

// StartSpan starts a new span within the trace
func (t *InMemoryTrace) StartSpan(ctx context.Context, name string, parentSpanID string) (context.Context, TraceSpan) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// Create a new span
	span := &InMemoryTraceSpan{
		id:        generateID(),
		name:      name,
		traceID:   t.id,
		parentID:  parentSpanID,
		startTime: time.Now(),
		events:    make([]TraceEvent, 0),
		tags:      make(map[string]string),
		trace:     t,
	}
	
	// Add the span to the trace
	t.spans[span.id] = span
	
	// Create a new context with the span
	newCtx := context.WithValue(ctx, spanContextKey, span)
	
	return newCtx, span
}

// ID returns the span ID
func (s *InMemoryTraceSpan) ID() string {
	return s.id
}

// Name returns the span name
func (s *InMemoryTraceSpan) Name() string {
	return s.name
}

// TraceID returns the parent trace ID
func (s *InMemoryTraceSpan) TraceID() string {
	return s.traceID
}

// ParentID returns the parent span ID
func (s *InMemoryTraceSpan) ParentID() string {
	return s.parentID
}

// StartTime returns the span start time
func (s *InMemoryTraceSpan) StartTime() time.Time {
	return s.startTime
}

// EndTime returns the span end time
func (s *InMemoryTraceSpan) EndTime() time.Time {
	return s.endTime
}

// AddEvent adds an event to the span
func (s *InMemoryTraceSpan) AddEvent(name string, attributes map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Create a new event
	event := TraceEvent{
		Name:       name,
		Attributes: attributes,
		Timestamp:  time.Now(),
	}
	
	// Add the event to the span
	s.events = append(s.events, event)
}

// AddTag adds a tag to the span
func (s *InMemoryTraceSpan) AddTag(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.tags[key] = value
}

// End ends the span
func (s *InMemoryTraceSpan) End() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.endTime = time.Now()
}

// GetTraceFromContext gets a trace from the context
func GetTraceFromContext(ctx context.Context) (Trace, bool) {
	trace, ok := ctx.Value(traceContextKey).(Trace)
	return trace, ok
}

// GetTraceSpanFromContext gets a trace span from the context
func GetTraceSpanFromContext(ctx context.Context) (TraceSpan, bool) {
	span, ok := ctx.Value(traceSpanContextKey).(TraceSpan)
	return span, ok
}

// traceContextKey is the key used to store traces in the context
type traceContextKeyType struct{}
var traceContextKey = traceContextKeyType{}

// traceSpanContextKey is the key used to store trace spans in the context
type traceSpanContextKeyType struct{}
var traceSpanContextKey = traceSpanContextKeyType{}

// Helper function to generate IDs
func generateID() string {
	return "id-" + time.Now().Format("20060102150405.999999999")
}

// ErrTraceNotFound is returned when a trace is not found
var ErrTraceNotFound = TraceError{
	Code:    "trace_not_found",
	Message: "Trace not found",
}

// TraceError represents an error when using tracing
type TraceError struct {
	Code    string
	Message string
}

// Error returns the error message
func (e TraceError) Error() string {
	return e.Message
}
