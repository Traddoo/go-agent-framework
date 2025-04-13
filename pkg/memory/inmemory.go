package memory

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryStore implements Interface with in-memory storage
type InMemoryStore struct {
	documents map[string]*Document
	chunks    map[string]*Chunk
	mutex     sync.RWMutex
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		documents: make(map[string]*Document),
		chunks:    make(map[string]*Chunk),
	}
}

// Initialize initializes the memory store
func (s *InMemoryStore) Initialize(ctx context.Context) error {
	// Nothing to initialize for in-memory store
	return nil
}

// CreateDocument creates a new document in the store
func (s *InMemoryStore) CreateDocument(ctx context.Context, doc *Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Generate an ID if not provided
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	
	// Check if document already exists
	if _, exists := s.documents[doc.ID]; exists {
		return errors.New("document already exists")
	}
	
	// Set creation and update times if not provided
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = doc.CreatedAt
	}
	
	// Set version if not provided
	if doc.Version == 0 {
		doc.Version = 1
	}
	
	// Initialize metadata if not provided
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	
	// Store the document
	s.documents[doc.ID] = doc
	
	return nil
}

// ReadDocument retrieves a document by ID
func (s *InMemoryStore) ReadDocument(ctx context.Context, id string) (*Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Check if document exists
	doc, exists := s.documents[id]
	if !exists {
		return nil, errors.New("document not found")
	}
	
	// Return a copy of the document to prevent modification
	docCopy := *doc
	return &docCopy, nil
}

// UpdateDocument updates an existing document
func (s *InMemoryStore) UpdateDocument(ctx context.Context, doc *Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if document exists
	existingDoc, exists := s.documents[doc.ID]
	if !exists {
		return errors.New("document not found")
	}
	
	// Check if document is locked by someone else
	if existingDoc.LockedBy != "" && existingDoc.LockedBy != doc.UpdatedBy && existingDoc.LockedUntil.After(time.Now()) {
		return errors.New("document is locked by another agent")
	}
	
	// Increment version
	doc.Version = existingDoc.Version + 1
	
	// Update timestamp
	doc.UpdatedAt = time.Now()
	
	// Store the updated document
	s.documents[doc.ID] = doc
	
	return nil
}

// DeleteDocument deletes a document by ID
func (s *InMemoryStore) DeleteDocument(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if document exists
	_, exists := s.documents[id]
	if !exists {
		return errors.New("document not found")
	}
	
	// Delete the document
	delete(s.documents, id)
	
	return nil
}

// ListDocuments lists documents matching the filter
func (s *InMemoryStore) ListDocuments(ctx context.Context, filter Filter) ([]*Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Create a slice to hold matching documents
	results := make([]*Document, 0)
	
	// Apply filters
	for _, doc := range s.documents {
		// Skip if doesn't match CreatedBy filter
		if filter.CreatedBy != "" && doc.CreatedBy != filter.CreatedBy {
			continue
		}
		
		// Skip if doesn't match UpdatedBy filter
		if filter.UpdatedBy != "" && doc.UpdatedBy != filter.UpdatedBy {
			continue
		}
		
		// Skip if doesn't match CreatedFrom filter
		if !filter.CreatedFrom.IsZero() && doc.CreatedAt.Before(filter.CreatedFrom) {
			continue
		}
		
		// Skip if doesn't match CreatedTo filter
		if !filter.CreatedTo.IsZero() && doc.CreatedAt.After(filter.CreatedTo) {
			continue
		}
		
		// Skip if doesn't match UpdatedFrom filter
		if !filter.UpdatedFrom.IsZero() && doc.UpdatedAt.Before(filter.UpdatedFrom) {
			continue
		}
		
		// Skip if doesn't match UpdatedTo filter
		if !filter.UpdatedTo.IsZero() && doc.UpdatedAt.After(filter.UpdatedTo) {
			continue
		}
		
		// Skip if doesn't have all required tags
		if len(filter.Tags) > 0 {
			hasAllTags := true
			for _, tag := range filter.Tags {
				found := false
				for _, docTag := range doc.Tags {
					if docTag == tag {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				continue
			}
		}
		
		// Add a copy of the document to results
		docCopy := *doc
		results = append(results, &docCopy)
	}
	
	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}
	
	return results, nil
}

// SearchDocuments searches for documents matching the query
func (s *InMemoryStore) SearchDocuments(ctx context.Context, query string) ([]*Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Create a slice to hold matching documents
	results := make([]*Document, 0)
	
	// Very basic search implementation - just check if query is contained in title or content
	// In a real implementation, this would use full-text search or vector search
	query = strings.ToLower(query)
	for _, doc := range s.documents {
		if strings.Contains(strings.ToLower(doc.Title), query) || strings.Contains(strings.ToLower(doc.Content), query) {
			// Add a copy of the document to results
			docCopy := *doc
			results = append(results, &docCopy)
		}
	}
	
	return results, nil
}

// Lock locks a document for exclusive access
func (s *InMemoryStore) Lock(ctx context.Context, id string, owner string, ttl time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if document exists
	doc, exists := s.documents[id]
	if !exists {
		return errors.New("document not found")
	}
	
	// Check if document is already locked by someone else
	if doc.LockedBy != "" && doc.LockedBy != owner && doc.LockedUntil.After(time.Now()) {
		return errors.New("document is already locked by another agent")
	}
	
	// Lock the document
	doc.LockedBy = owner
	doc.LockedUntil = time.Now().Add(ttl)
	
	return nil
}

// Unlock unlocks a document
func (s *InMemoryStore) Unlock(ctx context.Context, id string, owner string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if document exists
	doc, exists := s.documents[id]
	if !exists {
		return errors.New("document not found")
	}
	
	// Check if document is locked by someone else
	if doc.LockedBy != "" && doc.LockedBy != owner {
		return errors.New("document is locked by another agent")
	}
	
	// Unlock the document
	doc.LockedBy = ""
	doc.LockedUntil = time.Time{}
	
	return nil
}

// CreateChunk creates a memory chunk
func (s *InMemoryStore) CreateChunk(ctx context.Context, chunk *Chunk) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Generate an ID if not provided
	if chunk.ID == "" {
		chunk.ID = uuid.New().String()
	}
	
	// Check if chunk already exists
	if _, exists := s.chunks[chunk.ID]; exists {
		return errors.New("chunk already exists")
	}
	
	// Set creation and update times if not provided
	if chunk.CreatedAt.IsZero() {
		chunk.CreatedAt = time.Now()
	}
	if chunk.UpdatedAt.IsZero() {
		chunk.UpdatedAt = chunk.CreatedAt
	}
	
	// Initialize metadata if not provided
	if chunk.Metadata == nil {
		chunk.Metadata = make(map[string]interface{})
	}
	
	// Initialize embedding if not provided
	if chunk.Embedding == nil {
		// In a real implementation, this would compute an embedding for the content
		chunk.Embedding = make([]float32, 0)
	}
	
	// Store the chunk
	s.chunks[chunk.ID] = chunk
	
	return nil
}

// ReadChunk retrieves a chunk by ID
func (s *InMemoryStore) ReadChunk(ctx context.Context, id string) (*Chunk, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Check if chunk exists
	chunk, exists := s.chunks[id]
	if !exists {
		return nil, errors.New("chunk not found")
	}
	
	// Return a copy of the chunk to prevent modification
	chunkCopy := *chunk
	return &chunkCopy, nil
}

// UpdateChunk updates an existing chunk
func (s *InMemoryStore) UpdateChunk(ctx context.Context, chunk *Chunk) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if chunk exists
	_, exists := s.chunks[chunk.ID]
	if !exists {
		return errors.New("chunk not found")
	}
	
	// Update timestamp
	chunk.UpdatedAt = time.Now()
	
	// Store the updated chunk
	s.chunks[chunk.ID] = chunk
	
	return nil
}

// DeleteChunk deletes a chunk by ID
func (s *InMemoryStore) DeleteChunk(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if chunk exists
	_, exists := s.chunks[id]
	if !exists {
		return errors.New("chunk not found")
	}
	
	// Delete the chunk
	delete(s.chunks, id)
	
	return nil
}

// SearchChunks searches for chunks semantically similar to the query
func (s *InMemoryStore) SearchChunks(ctx context.Context, query string, maxResults int) ([]*Chunk, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Create a slice to hold matching chunks
	results := make([]*Chunk, 0)
	
	// Very basic search implementation - just check if query is contained in content
	// In a real implementation, this would compute an embedding for the query and find similar chunks
	query = strings.ToLower(query)
	for _, chunk := range s.chunks {
		if strings.Contains(strings.ToLower(chunk.Content), query) {
			// Add a copy of the chunk to results
			chunkCopy := *chunk
			results = append(results, &chunkCopy)
		}
	}
	
	// Limit results
	if maxResults > 0 && maxResults < len(results) {
		results = results[:maxResults]
	}
	
	return results, nil
}
