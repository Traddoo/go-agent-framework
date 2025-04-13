package comms

import (
	"encoding/json"
	"time"
)

// Message represents a communication message between agents
type Message struct {
	ID        string
	From      string
	To        string
	Topic     string
	Type      string
	Content   interface{}
	Headers   map[string]string
	Timestamp time.Time
}

// NewMessage creates a new message
func NewMessage(from string, to string, messageType string, content interface{}) *Message {
	return &Message{
		ID:        generateMessageID(),
		From:      from,
		To:        to,
		Type:      messageType,
		Content:   content,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

// SetHeader sets a header on the message
func (m *Message) SetHeader(key string, value string) {
	m.Headers[key] = value
}

// GetHeader gets a header from the message
func (m *Message) GetHeader(key string) (string, bool) {
	value, exists := m.Headers[key]
	return value, exists
}

// ContentAsMap returns the content as a map
func (m *Message) ContentAsMap() (map[string]interface{}, error) {
	// If it's already a map, return it
	if contentMap, ok := m.Content.(map[string]interface{}); ok {
		return contentMap, nil
	}
	
	// Convert to JSON and back to get a map
	contentBytes, err := json.Marshal(m.Content)
	if err != nil {
		return nil, err
	}
	
	var contentMap map[string]interface{}
	err = json.Unmarshal(contentBytes, &contentMap)
	if err != nil {
		return nil, err
	}
	
	return contentMap, nil
}

// ContentAsString returns the content as a string
func (m *Message) ContentAsString() (string, error) {
	// If it's already a string, return it
	if contentStr, ok := m.Content.(string); ok {
		return contentStr, nil
	}
	
	// Convert to JSON to get a string
	contentBytes, err := json.Marshal(m.Content)
	if err != nil {
		return "", err
	}
	
	return string(contentBytes), nil
}

// MessageTypes defines common message types
var MessageTypes = struct {
	Request  string
	Response string
	Event    string
	Command  string
	Status   string
}{
	Request:  "request",
	Response: "response",
	Event:    "event",
	Command:  "command",
	Status:   "status",
}

// Common headers
var Headers = struct {
	ContentType    string
	CorrelationID  string
	Priority       string
	TTL            string
	Timestamp      string
	Error          string
	ErrorCode      string
	ErrorMessage   string
}{
	ContentType:    "content_type",
	CorrelationID:  "correlation_id",
	Priority:       "priority",
	TTL:            "ttl",
	Timestamp:      "timestamp",
	Error:          "error",
	ErrorCode:      "error_code",
	ErrorMessage:   "error_message",
}

// Content types
var ContentTypes = struct {
	JSON      string
	Text      string
	Binary    string
	Document  string
	Task      string
	Result    string
}{
	JSON:      "application/json",
	Text:      "text/plain",
	Binary:    "application/octet-stream",
	Document:  "application/document",
	Task:      "application/task",
	Result:    "application/result",
}

