package memory

import (
	"context"
	"time"
)

// Interface defines the common interface for memory/document storage
type Interface interface {
	// Initialize initializes the memory store
	Initialize(ctx context.Context) error
	
	// CreateDocument creates a new document in the store
	CreateDocument(ctx context.Context, doc *Document) error
	
	// ReadDocument retrieves a document by ID
	ReadDocument(ctx context.Context, id string) (*Document, error)
	
	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, doc *Document) error
	
	// DeleteDocument deletes a document by ID
	DeleteDocument(ctx context.Context, id string) error
	
	// ListDocuments lists documents matching the filter
	ListDocuments(ctx context.Context, filter Filter) ([]*Document, error)
	
	// SearchDocuments searches for documents matching the query
	SearchDocuments(ctx context.Context, query string) ([]*Document, error)
	
	// Lock locks a document for exclusive access
	Lock(ctx context.Context, id string, owner string, ttl time.Duration) error
	
	// Unlock unlocks a document
	Unlock(ctx context.Context, id string, owner string) error
	
	// CreateChunk creates a memory chunk
	CreateChunk(ctx context.Context, chunk *Chunk) error
	
	// ReadChunk retrieves a chunk by ID
	ReadChunk(ctx context.Context, id string) (*Chunk, error)
	
	// UpdateChunk updates an existing chunk
	UpdateChunk(ctx context.Context, chunk *Chunk) error
	
	// DeleteChunk deletes a chunk by ID
	DeleteChunk(ctx context.Context, id string) error
	
	// SearchChunks searches for chunks semantically similar to the query
	SearchChunks(ctx context.Context, query string, maxResults int) ([]*Chunk, error)
}

// Document represents a document in the memory store
type Document struct {
	ID          string
	Title       string
	Content     string
	Version     int64
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Tags        []string
	Metadata    map[string]interface{}
	LockedBy    string
	LockedUntil time.Time
}

// Chunk represents a smaller piece of memory for semantic retrieval
type Chunk struct {
	ID          string
	Content     string
	Metadata    map[string]interface{}
	Embedding   []float32
	CreatedAt   time.Time
	UpdatedAt   time.Time
	SourceID    string  // Optional reference to source document
}

// Filter represents a filter for listing documents
type Filter struct {
	Tags        []string
	CreatedBy   string
	UpdatedBy   string
	CreatedFrom time.Time
	CreatedTo   time.Time
	UpdatedFrom time.Time
	UpdatedTo   time.Time
	Limit       int
	Offset      int
}
