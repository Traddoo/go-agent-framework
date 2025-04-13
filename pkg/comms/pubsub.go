package comms

import (
	"context"
	"errors"
	"sync"
	"time"
)

// PubSub implements a publish-subscribe pattern
type PubSub struct {
	topics          map[string][]*Subscription
	mutex           sync.RWMutex
	defaultTimeout  time.Duration
}

// Subscription represents a subscription to a topic
type Subscription struct {
	ID        string
	Topic     string
	Handler   MessageHandler
	Active    bool
	CreatedAt time.Time
}

// NewPubSub creates a new pub/sub system
func NewPubSub(defaultTimeout time.Duration) *PubSub {
	if defaultTimeout == 0 {
		defaultTimeout = 30 * time.Second
	}
	
	return &PubSub{
		topics:         make(map[string][]*Subscription),
		defaultTimeout: defaultTimeout,
	}
}

// Publish publishes a message to a topic
func (ps *PubSub) Publish(ctx context.Context, topic string, message *Message) error {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Check if there are any subscribers
	subscriptions, exists := ps.topics[topic]
	if !exists || len(subscriptions) == 0 {
		return nil
	}
	
	// Set the topic and timestamp on the message
	message.Topic = topic
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	
	// Deliver the message to all active subscribers
	for _, subscription := range subscriptions {
		if subscription.Active {
			go func(sub *Subscription) {
				// Create a context with timeout
				deliveryCtx, cancel := context.WithTimeout(ctx, ps.defaultTimeout)
				defer cancel()
				
				// Call the handler
				sub.Handler(deliveryCtx, message)
			}(subscription)
		}
	}
	
	return nil
}

// Subscribe subscribes to a topic
func (ps *PubSub) Subscribe(ctx context.Context, topic string, handler MessageHandler) (*Subscription, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Create the subscription
	subscription := &Subscription{
		ID:        generateSubscriptionID(topic),
		Topic:     topic,
		Handler:   handler,
		Active:    true,
		CreatedAt: time.Now(),
	}
	
	// Add the subscription to the topic
	ps.topics[topic] = append(ps.topics[topic], subscription)
	
	return subscription, nil
}

// Unsubscribe unsubscribes from a topic
func (ps *PubSub) Unsubscribe(ctx context.Context, subscriptionID string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Extract the topic from the subscription ID
	topic, err := extractTopicFromSubscriptionID(subscriptionID)
	if err != nil {
		return err
	}
	
	// Find the subscription
	subscriptions, exists := ps.topics[topic]
	if !exists {
		return errors.New("topic not found")
	}
	
	for i, subscription := range subscriptions {
		if subscription.ID == subscriptionID {
			// Mark the subscription as inactive
			subscription.Active = false
			
			// Remove the subscription from the slice
			ps.topics[topic] = append(subscriptions[:i], subscriptions[i+1:]...)
			
			// Remove the topic if there are no more subscriptions
			if len(ps.topics[topic]) == 0 {
				delete(ps.topics, topic)
			}
			
			return nil
		}
	}
	
	return errors.New("subscription not found")
}

// CreateTopic creates a new topic
func (ps *PubSub) CreateTopic(ctx context.Context, topic string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Check if the topic already exists
	if _, exists := ps.topics[topic]; exists {
		return errors.New("topic already exists")
	}
	
	// Create the topic
	ps.topics[topic] = make([]*Subscription, 0)
	
	return nil
}

// DeleteTopic deletes a topic
func (ps *PubSub) DeleteTopic(ctx context.Context, topic string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// Check if the topic exists
	if _, exists := ps.topics[topic]; !exists {
		return errors.New("topic not found")
	}
	
	// Delete the topic
	delete(ps.topics, topic)
	
	return nil
}

// ListTopics lists all topics
func (ps *PubSub) ListTopics(ctx context.Context) []string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Create a slice with all topics
	topics := make([]string, 0, len(ps.topics))
	for topic := range ps.topics {
		topics = append(topics, topic)
	}
	
	return topics
}

// ListSubscriptions lists all subscriptions for a topic
func (ps *PubSub) ListSubscriptions(ctx context.Context, topic string) ([]*Subscription, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Check if the topic exists
	subscriptions, exists := ps.topics[topic]
	if !exists {
		return nil, errors.New("topic not found")
	}
	
	// Create a slice with all subscriptions
	result := make([]*Subscription, len(subscriptions))
	copy(result, subscriptions)
	
	return result, nil
}

// CountSubscriptions counts the number of subscriptions for a topic
func (ps *PubSub) CountSubscriptions(ctx context.Context, topic string) (int, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Check if the topic exists
	subscriptions, exists := ps.topics[topic]
	if !exists {
		return 0, errors.New("topic not found")
	}
	
	return len(subscriptions), nil
}

// CountActiveSubscriptions counts the number of active subscriptions for a topic
func (ps *PubSub) CountActiveSubscriptions(ctx context.Context, topic string) (int, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	// Check if the topic exists
	subscriptions, exists := ps.topics[topic]
	if !exists {
		return 0, errors.New("topic not found")
	}
	
	// Count active subscriptions
	count := 0
	for _, subscription := range subscriptions {
		if subscription.Active {
			count++
		}
	}
	
	return count, nil
}
