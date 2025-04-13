package comms

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Bus defines the communication bus interface
type Bus interface {
	// Initialize initializes the communication bus
	Initialize(ctx context.Context) error
	
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, message *Message) error
	
	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topic string, handler MessageHandler) (string, error)
	
	// Unsubscribe unsubscribes from a topic
	Unsubscribe(ctx context.Context, subscriptionID string) error
	
	// RequestResponse sends a request and waits for a response
	RequestResponse(ctx context.Context, target string, request *Message, timeout time.Duration) (*Message, error)
}

// MessageHandler is a function that handles messages
type MessageHandler func(ctx context.Context, message *Message) error

// InMemoryBus implements the Bus interface with in-memory storage
type InMemoryBus struct {
	subscriptions     map[string]map[string]MessageHandler
	requestHandlers   map[string]MessageHandler
	subscriptionMutex sync.RWMutex
	requestMutex      sync.RWMutex
}

// NewInMemoryBus creates a new in-memory communication bus
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		subscriptions:   make(map[string]map[string]MessageHandler),
		requestHandlers: make(map[string]MessageHandler),
	}
}

// Initialize initializes the communication bus
func (b *InMemoryBus) Initialize(ctx context.Context) error {
	// Nothing to initialize for in-memory bus
	return nil
}

// Publish publishes a message to a topic
func (b *InMemoryBus) Publish(ctx context.Context, topic string, message *Message) error {
	b.subscriptionMutex.RLock()
	defer b.subscriptionMutex.RUnlock()
	
	// Check if there are any subscribers for this topic
	subscribers, exists := b.subscriptions[topic]
	if !exists || len(subscribers) == 0 {
		return nil
	}
	
	// Set the topic and timestamp on the message
	message.Topic = topic
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	
	// Call each subscriber handler
	for _, handler := range subscribers {
		go handler(ctx, message)
	}
	
	return nil
}

// Subscribe subscribes to a topic
func (b *InMemoryBus) Subscribe(ctx context.Context, topic string, handler MessageHandler) (string, error) {
	b.subscriptionMutex.Lock()
	defer b.subscriptionMutex.Unlock()
	
	// Initialize the topic if it doesn't exist
	if _, exists := b.subscriptions[topic]; !exists {
		b.subscriptions[topic] = make(map[string]MessageHandler)
	}
	
	// Generate a subscription ID
	subscriptionID := generateSubscriptionID(topic)
	
	// Add the handler
	b.subscriptions[topic][subscriptionID] = handler
	
	return subscriptionID, nil
}

// Unsubscribe unsubscribes from a topic
func (b *InMemoryBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	b.subscriptionMutex.Lock()
	defer b.subscriptionMutex.Unlock()
	
	// Extract the topic from the subscription ID
	topic, err := extractTopicFromSubscriptionID(subscriptionID)
	if err != nil {
		return err
	}
	
	// Check if the topic exists
	subscribers, exists := b.subscriptions[topic]
	if !exists {
		return errors.New("topic not found")
	}
	
	// Check if the subscription exists
	if _, exists := subscribers[subscriptionID]; !exists {
		return errors.New("subscription not found")
	}
	
	// Remove the subscription
	delete(subscribers, subscriptionID)
	
	// Remove the topic if there are no more subscribers
	if len(subscribers) == 0 {
		delete(b.subscriptions, topic)
	}
	
	return nil
}

// RegisterRequestHandler registers a handler for request-response messages
func (b *InMemoryBus) RegisterRequestHandler(ctx context.Context, target string, handler MessageHandler) error {
	b.requestMutex.Lock()
	defer b.requestMutex.Unlock()
	
	// Check if the target already has a handler
	if _, exists := b.requestHandlers[target]; exists {
		return errors.New("target already has a handler")
	}
	
	// Register the handler
	b.requestHandlers[target] = handler
	
	return nil
}

// UnregisterRequestHandler unregisters a request handler
func (b *InMemoryBus) UnregisterRequestHandler(ctx context.Context, target string) error {
	b.requestMutex.Lock()
	defer b.requestMutex.Unlock()
	
	// Check if the target has a handler
	if _, exists := b.requestHandlers[target]; !exists {
		return errors.New("target has no handler")
	}
	
	// Remove the handler
	delete(b.requestHandlers, target)
	
	return nil
}

// RequestResponse sends a request and waits for a response
func (b *InMemoryBus) RequestResponse(ctx context.Context, target string, request *Message, timeout time.Duration) (*Message, error) {
	b.requestMutex.RLock()
	
	// Check if the target has a handler
	handler, exists := b.requestHandlers[target]
	b.requestMutex.RUnlock()
	
	if !exists {
		return nil, errors.New("no handler for target")
	}
	
	// Set the target and timestamp on the message
	request.To = target
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}
	
	// Create a channel for the response
	responseCh := make(chan *Message, 1)
	errCh := make(chan error, 1)
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Call the handler in a goroutine
	go func() {
		// Create a response message
		response := &Message{
			ID:        generateMessageID(),
			From:      target,
			To:        request.From,
			Timestamp: time.Now(),
			Type:      "response",
			Headers:   make(map[string]string),
		}
		
		// Add the correlation ID
		response.Headers["correlation_id"] = request.ID
		
		// Call the handler
		err := handler(ctx, request)
		if err != nil {
			// Set the error in the response
			response.Headers["error"] = err.Error()
			errCh <- err
		}
		
		// Send the response
		responseCh <- response
	}()
	
	// Wait for the response or timeout
	select {
	case response := <-responseCh:
		return response, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Helper functions for subscription IDs and message IDs
func generateSubscriptionID(topic string) string {
	return topic + ":" + generateID()
}

func extractTopicFromSubscriptionID(subscriptionID string) (string, error) {
	// In a real implementation, this would parse the subscription ID
	// For this example, we'll return a placeholder
	return "topic", nil
}

func generateMessageID() string {
	// In a real implementation, this would generate a UUID
	return "msg-" + generateID()
}

func generateID() string {
	// In a real implementation, this would generate a UUID
	return "id-" + time.Now().Format("20060102150405")
}
