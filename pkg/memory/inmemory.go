package memory

import (
	"context"
	"errors"
	"math"
	"sort"
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
	
	// Generate embedding if not provided
	if chunk.Embedding == nil {
		// TODO: Replace with actual embedding generation
		// In a production system, we would call an embedding model API
		// For example, OpenAI's text-embedding-3-small or text-embedding-3-large
		
		// For now, we'll create a mock embedding
		// This is just a placeholder - real embeddings would be semantic vectors
		embedding := generateMockEmbedding(chunk.Content)
		chunk.Embedding = embedding
	}
	
	// Store the chunk
	s.chunks[chunk.ID] = chunk
	
	return nil
}

// generateMockEmbedding creates a simple mock embedding from text
// This is just for demonstration - real systems would use embeddings from models
func generateMockEmbedding(text string) []float32 {
	// Create a fixed-size embedding (real ones would be 768-1536 dimensions)
	embedding := make([]float32, 128)
	
	// Very simplistic approach - hash characters to positions
	for i, char := range text {
		pos := i % len(embedding)
		embedding[pos] += float32(char) / 1000
	}
	
	// Normalize the embedding (to unit vector)
	var sum float32
	for _, val := range embedding {
		sum += val * val
	}
	magnitude := float32(math.Sqrt(float64(sum)))
	
	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}
	
	return embedding
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
	
	// Generate an embedding for the query
	queryEmbedding := generateMockEmbedding(query)
	
	// Create a map to store similarity scores using cosine similarity
	scores := make(map[string]float64)
	
	// Calculate cosine similarity for each chunk's embedding
	for id, chunk := range s.chunks {
		if chunk.Embedding == nil || len(chunk.Embedding) == 0 {
			// Skip chunks without embeddings
			continue
		}
		
		// Calculate cosine similarity between query embedding and chunk embedding
		similarity := cosineSimilarity(queryEmbedding, chunk.Embedding)
		
		// Store scores above a threshold
		if similarity > 0.3 { // Arbitrary threshold
			scores[id] = similarity
		}
	}
	
	// If we found no semantic matches, fall back to text matching
	if len(scores) == 0 {
		// Basic text matching as fallback
		query = strings.ToLower(query)
		
		for id, chunk := range s.chunks {
			content := strings.ToLower(chunk.Content)
			
			var score float64
			
			if content == query {
				score = 1.0
			} else if strings.Contains(content, query) {
				score = float64(len(query)) / float64(len(content))
			} else {
				// Check for word overlaps
				queryWords := strings.Fields(query)
				contentWords := strings.Fields(content)
				
				// Create maps to count word frequency
				queryWordCount := make(map[string]int)
				for _, word := range queryWords {
					queryWordCount[word]++
				}
				
				// Count matching words
				matchCount := 0
				for _, word := range contentWords {
					if count, exists := queryWordCount[word]; exists && count > 0 {
						matchCount++
						queryWordCount[word]--
					}
				}
				
				if len(queryWords) > 0 {
					score = float64(matchCount) / float64(len(queryWords))
				}
			}
			
			// Store the score
			if score > 0 {
				scores[id] = score
			}
		}
	}
	
	// Sort chunks by score
	type ScoredChunk struct {
		Chunk *Chunk
		Score float64
	}
	
	scoredChunks := make([]ScoredChunk, 0, len(scores))
	for id, score := range scores {
		chunk := s.chunks[id]
		scoredChunks = append(scoredChunks, ScoredChunk{
			Chunk: chunk,
			Score: score,
		})
	}
	
	// Sort by score (descending)
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})
	
	// Take top N results
	for i, sc := range scoredChunks {
		if maxResults > 0 && i >= maxResults {
			break
		}
		
		// Add a copy of the chunk to results
		chunkCopy := *sc.Chunk
		results = append(results, &chunkCopy)
	}
	
	return results, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	
	var dotProduct float64
	var magnitudeA float64
	var magnitudeB float64
	
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		magnitudeA += float64(a[i]) * float64(a[i])
		magnitudeB += float64(b[i]) * float64(b[i])
	}
	
	magnitudeA = math.Sqrt(magnitudeA)
	magnitudeB = math.Sqrt(magnitudeB)
	
	if magnitudeA == 0 || magnitudeB == 0 {
		return 0
	}
	
	return dotProduct / (magnitudeA * magnitudeB)
}
