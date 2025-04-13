package cognitive

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/traddoo/go-agent-framework/pkg/memory"
)

// CognitiveMemory implements a human-like memory system with working, short-term,
// and long-term memory components
type CognitiveMemory struct {
	// Internal memory store for persisting long-term memories
	store           memory.Interface
	
	// Working memory (scratchpad) - very limited, highly accessible
	workingMemory   []*MemoryItem
	workingCapacity int
	
	// Short-term memory - recent items, decays over time
	shortTermMemory []*MemoryItem 
	shortTermCapacity int
	
	// Long-term memory is stored in the memory.Interface
	// But we keep references to recently accessed items
	recentlyAccessed []*memory.Chunk
	
	// Importance threshold for moving to long-term memory
	importanceThreshold float64
	
	// Mutex for thread safety
	mu              sync.RWMutex
	
	// Agent ID to track memories by agent
	agentID         string
}

// MemoryItem represents an item in working or short-term memory
type MemoryItem struct {
	ID          string
	Content     string
	Created     time.Time
	LastAccess  time.Time
	AccessCount int
	Importance  float64
	Type        MemoryType
	Metadata    map[string]interface{}
}

// MemoryType defines the types of memories
type MemoryType string

const (
	WorkingMemory   MemoryType = "working"
	ShortTermMemory MemoryType = "short-term"
	LongTermMemory  MemoryType = "long-term"
	EpisodicMemory  MemoryType = "episodic"    // experiences and events
	SemanticMemory  MemoryType = "semantic"    // facts and concepts
	ProceduralMemory MemoryType = "procedural" // skills and methods
)

// NewCognitiveMemory creates a new cognitive memory system
func NewCognitiveMemory(store memory.Interface, agentID string) *CognitiveMemory {
	return &CognitiveMemory{
		store:             store,
		workingMemory:     make([]*MemoryItem, 0),
		workingCapacity:   7,                 // Miller's Law: 7±2 items
		shortTermMemory:   make([]*MemoryItem, 0),
		shortTermCapacity: 20,
		recentlyAccessed:  make([]*memory.Chunk, 0),
		importanceThreshold: 0.5,
		agentID:           agentID,
	}
}

// AddToWorkingMemory adds an item to working memory
func (cm *CognitiveMemory) AddToWorkingMemory(content string, importance float64, metadata map[string]interface{}) (*MemoryItem, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Create a new memory item
	item := &MemoryItem{
		ID:          uuid.New().String(),
		Content:     content,
		Created:     time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
		Importance:  importance,
		Type:        WorkingMemory,
		Metadata:    metadata,
	}
	
	// If working memory is full, move the least important item to short-term memory
	if len(cm.workingMemory) >= cm.workingCapacity {
		cm.consolidateWorkingMemory()
	}
	
	// Add the item to working memory
	cm.workingMemory = append(cm.workingMemory, item)
	
	return item, nil
}

// consolidateWorkingMemory moves the least important items from working to short-term memory
func (cm *CognitiveMemory) consolidateWorkingMemory() {
	if len(cm.workingMemory) == 0 {
		return
	}
	
	// Sort by importance and recency (last access time)
	sort.Slice(cm.workingMemory, func(i, j int) bool {
		// If importance difference is significant
		if cm.workingMemory[i].Importance != cm.workingMemory[j].Importance {
			return cm.workingMemory[i].Importance < cm.workingMemory[j].Importance
		}
		// Otherwise sort by recency (oldest first)
		return cm.workingMemory[i].LastAccess.Before(cm.workingMemory[j].LastAccess)
	})
	
	// Move the least important/oldest item to short-term memory
	item := cm.workingMemory[0]
	item.Type = ShortTermMemory
	
	// Remove from working memory
	cm.workingMemory = cm.workingMemory[1:]
	
	// Add to short-term memory
	if len(cm.shortTermMemory) >= cm.shortTermCapacity {
		cm.consolidateShortTermMemory()
	}
	
	cm.shortTermMemory = append(cm.shortTermMemory, item)
}

// consolidateShortTermMemory moves important items from short-term to long-term memory
func (cm *CognitiveMemory) consolidateShortTermMemory() {
	if len(cm.shortTermMemory) == 0 {
		return
	}
	
	// Sort by importance and recency (least important and oldest first)
	sort.Slice(cm.shortTermMemory, func(i, j int) bool {
		// If importance difference is significant
		if cm.shortTermMemory[i].Importance != cm.shortTermMemory[j].Importance {
			return cm.shortTermMemory[i].Importance < cm.shortTermMemory[j].Importance
		}
		// Otherwise sort by recency (oldest first)
		return cm.shortTermMemory[i].LastAccess.Before(cm.shortTermMemory[j].LastAccess)
	})
	
	// Check if the most important item should be moved to long-term memory
	if cm.shortTermMemory[len(cm.shortTermMemory)-1].Importance >= cm.importanceThreshold {
		item := cm.shortTermMemory[len(cm.shortTermMemory)-1]
		
		// Move to long-term memory (as a chunk)
		chunk := &memory.Chunk{
			ID:      fmt.Sprintf("memory-%s", item.ID),
			Content: item.Content,
			Metadata: map[string]interface{}{
				"agent_id":    cm.agentID,
				"importance":  item.Importance,
				"memory_type": string(item.Type),
				"created":     item.Created.Format(time.RFC3339),
				"access_count": item.AccessCount,
			},
			CreatedAt: item.Created,
			UpdatedAt: time.Now(),
		}
		
		// Add user metadata
		for k, v := range item.Metadata {
			chunk.Metadata[k] = v
		}
		
		// Store in long-term memory
		// Ignore errors for now
		_ = cm.store.CreateChunk(context.Background(), chunk)
		
		// Remove from short-term memory
		cm.shortTermMemory = cm.shortTermMemory[:len(cm.shortTermMemory)-1]
	}
	
	// Discard the least important item if we're still over capacity
	if len(cm.shortTermMemory) >= cm.shortTermCapacity {
		cm.shortTermMemory = cm.shortTermMemory[1:]
	}
}

// Recall searches all memory systems for information
func (cm *CognitiveMemory) Recall(ctx context.Context, query string, memoryTypes []MemoryType) ([]interface{}, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	var results []interface{}
	
	// Check if we should search each memory type
	searchWorking := false
	searchShortTerm := false
	searchLongTerm := false
	
	for _, mt := range memoryTypes {
		switch mt {
		case WorkingMemory:
			searchWorking = true
		case ShortTermMemory:
			searchShortTerm = true
		case LongTermMemory, EpisodicMemory, SemanticMemory, ProceduralMemory:
			searchLongTerm = true
		}
	}
	
	// Search working memory (highest priority, checked first)
	if searchWorking {
		for _, item := range cm.workingMemory {
			// Simple string matching for now
			// In a real implementation, use more sophisticated matching
			if strings.Contains(strings.ToLower(item.Content), strings.ToLower(query)) {
				// Update access info
				item.LastAccess = time.Now()
				item.AccessCount++
				
				results = append(results, item)
			}
		}
	}
	
	// Search short-term memory (medium priority)
	if searchShortTerm {
		for _, item := range cm.shortTermMemory {
			// Simple string matching for now
			if strings.Contains(strings.ToLower(item.Content), strings.ToLower(query)) {
				// Update access info
				item.LastAccess = time.Now()
				item.AccessCount++
				
				// If accessed frequently, consider moving back to working memory
				if item.AccessCount > 3 {
					item.Type = WorkingMemory
					cm.workingMemory = append(cm.workingMemory, item)
					
					// Remove from short-term
					for i, stm := range cm.shortTermMemory {
						if stm.ID == item.ID {
							cm.shortTermMemory = append(cm.shortTermMemory[:i], cm.shortTermMemory[i+1:]...)
							break
						}
					}
					
					// Consolidate if necessary
					if len(cm.workingMemory) > cm.workingCapacity {
						cm.consolidateWorkingMemory()
					}
				}
				
				results = append(results, item)
			}
		}
	}
	
	// Search long-term memory (lowest priority but largest capacity)
	if searchLongTerm {
		// Filter for specific memory types if needed
		var memoryTypeStrings []string
		for _, mt := range memoryTypes {
			if mt == LongTermMemory || mt == EpisodicMemory || mt == SemanticMemory || mt == ProceduralMemory {
				memoryTypeStrings = append(memoryTypeStrings, string(mt))
			}
		}
		
		// Use vector search for long-term memory
		chunks, err := cm.store.SearchChunks(ctx, query, 10)
		if err != nil {
			return results, fmt.Errorf("error searching long-term memory: %w", err)
		}
		
		// Filter chunks by agent and memory type if specified
		for _, chunk := range chunks {
			// Check if this chunk belongs to this agent
			agentID, ok := chunk.Metadata["agent_id"].(string)
			if !ok || agentID != cm.agentID {
				continue
			}
			
			// Check if memory type matches what we're looking for
			if len(memoryTypeStrings) > 0 {
				memType, ok := chunk.Metadata["memory_type"].(string)
				if !ok {
					continue
				}
				
				matchesType := false
				for _, mt := range memoryTypeStrings {
					if memType == mt {
						matchesType = true
						break
					}
				}
				
				if !matchesType {
					continue
				}
			}
			
			// Track recently accessed
			cm.recentlyAccessed = append(cm.recentlyAccessed, chunk)
			if len(cm.recentlyAccessed) > 10 {
				cm.recentlyAccessed = cm.recentlyAccessed[1:]
			}
			
			results = append(results, chunk)
		}
	}
	
	return results, nil
}

// GetWorkingMemory returns all items in working memory
func (cm *CognitiveMemory) GetWorkingMemory() []*MemoryItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make([]*MemoryItem, len(cm.workingMemory))
	copy(result, cm.workingMemory)
	
	return result
}

// GetShortTermMemory returns all items in short-term memory
func (cm *CognitiveMemory) GetShortTermMemory() []*MemoryItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make([]*MemoryItem, len(cm.shortTermMemory))
	copy(result, cm.shortTermMemory)
	
	return result
}

// Reflect strengthens important memories by increasing their importance
// and accelerating their transfer to long-term memory
func (cm *CognitiveMemory) Reflect(ctx context.Context, memoryIDs []string, importanceIncrease float64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Map to track which items we've found
	found := make(map[string]bool)
	
	// Check working memory
	for _, item := range cm.workingMemory {
		for _, id := range memoryIDs {
			if item.ID == id {
				item.Importance += importanceIncrease
				if item.Importance > 1.0 {
					item.Importance = 1.0
				}
				found[id] = true
			}
		}
	}
	
	// Check short-term memory
	for _, item := range cm.shortTermMemory {
		for _, id := range memoryIDs {
			if item.ID == id {
				item.Importance += importanceIncrease
				if item.Importance > 1.0 {
					item.Importance = 1.0
				}
				found[id] = true
				
				// If importance is now above threshold, move to long-term memory
				if item.Importance >= cm.importanceThreshold {
					// Move to long-term memory (as a chunk)
					chunk := &memory.Chunk{
						ID:      fmt.Sprintf("memory-%s", item.ID),
						Content: item.Content,
						Metadata: map[string]interface{}{
							"agent_id":    cm.agentID,
							"importance":  item.Importance,
							"memory_type": string(item.Type),
							"created":     item.Created.Format(time.RFC3339),
							"access_count": item.AccessCount,
						},
						CreatedAt: item.Created,
						UpdatedAt: time.Now(),
					}
					
					// Add user metadata
					for k, v := range item.Metadata {
						chunk.Metadata[k] = v
					}
					
					// Store in long-term memory
					err := cm.store.CreateChunk(ctx, chunk)
					if err != nil {
						return fmt.Errorf("error storing memory in long-term storage: %w", err)
					}
					
					// Remove from short-term memory
					for i, stm := range cm.shortTermMemory {
						if stm.ID == item.ID {
							cm.shortTermMemory = append(cm.shortTermMemory[:i], cm.shortTermMemory[i+1:]...)
							break
						}
					}
				}
			}
		}
	}
	
	// Check if we found all the memories
	for _, id := range memoryIDs {
		if !found[id] {
			// Check if it might be in long-term memory
			// In a real implementation, you would search by ID
			// For now, we'll skip this step
		}
	}
	
	return nil
}

// Forget removes items from memory (deliberate forgetting)
func (cm *CognitiveMemory) Forget(ctx context.Context, memoryIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Remove from working memory
	newWorking := make([]*MemoryItem, 0, len(cm.workingMemory))
	for _, item := range cm.workingMemory {
		shouldKeep := true
		for _, id := range memoryIDs {
			if item.ID == id {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			newWorking = append(newWorking, item)
		}
	}
	cm.workingMemory = newWorking
	
	// Remove from short-term memory
	newShortTerm := make([]*MemoryItem, 0, len(cm.shortTermMemory))
	for _, item := range cm.shortTermMemory {
		shouldKeep := true
		for _, id := range memoryIDs {
			if item.ID == id {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			newShortTerm = append(newShortTerm, item)
		}
	}
	cm.shortTermMemory = newShortTerm
	
	// For long-term memory, we'd typically mark items as "forgotten"
	// rather than actually deleting them, but that's implementation-specific
	
	return nil
}

// MaintainMemories performs routine maintenance like:
// - Decay of short-term memories over time
// - Consolidation of important memories to long-term
// - Cleanup of working memory
func (cm *CognitiveMemory) MaintainMemories(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	now := time.Now()
	
	// Decay short-term memories (reduce importance over time)
	for i := 0; i < len(cm.shortTermMemory); i++ {
		item := cm.shortTermMemory[i]
		
		// Calculate time since last access
		elapsed := now.Sub(item.LastAccess)
		
		// Apply decay based on time (more decay for older memories)
		// Simple linear decay for now
		decayFactor := elapsed.Hours() / 24.0 // % per day
		item.Importance -= decayFactor * 0.1  // 10% per day
		
		// Remove if importance is too low
		if item.Importance <= 0.1 {
			// Remove this item
			cm.shortTermMemory = append(cm.shortTermMemory[:i], cm.shortTermMemory[i+1:]...)
			i-- // Adjust index since we removed an item
		}
	}
	
	// Consolidate important short-term memories to long-term
	// Find items above the importance threshold
	var itemsToMove []*MemoryItem
	for _, item := range cm.shortTermMemory {
		if item.Importance >= cm.importanceThreshold {
			itemsToMove = append(itemsToMove, item)
		}
	}
	
	// Move them to long-term memory
	for _, item := range itemsToMove {
		// Move to long-term memory (as a chunk)
		chunk := &memory.Chunk{
			ID:      fmt.Sprintf("memory-%s", item.ID),
			Content: item.Content,
			Metadata: map[string]interface{}{
				"agent_id":     cm.agentID,
				"importance":   item.Importance,
				"memory_type":  string(item.Type),
				"created":      item.Created.Format(time.RFC3339),
				"access_count": item.AccessCount,
			},
			CreatedAt: item.Created,
			UpdatedAt: time.Now(),
		}
		
		// Add user metadata
		for k, v := range item.Metadata {
			chunk.Metadata[k] = v
		}
		
		// Store in long-term memory (ignoring errors for maintenance)
		_ = cm.store.CreateChunk(ctx, chunk)
		
		// Remove from short-term memory
		for i, stm := range cm.shortTermMemory {
			if stm.ID == item.ID {
				cm.shortTermMemory = append(cm.shortTermMemory[:i], cm.shortTermMemory[i+1:]...)
				break
			}
		}
	}
	
	return nil
}