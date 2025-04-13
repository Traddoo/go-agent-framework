package main

import (
	"context"
	"fmt"
	"time"

	"github.com/traddoo/go-agent-framework/pkg/memory"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create a memory store
	store := memory.NewInMemoryStore()
	if err := store.Initialize(ctx); err != nil {
		fmt.Printf("Error initializing memory store: %v\n", err)
		return
	}

	fmt.Println("=== Testing Document Operations ===")
	
	// Create a document
	doc := &memory.Document{
		ID:        "doc-1",
		Title:     "Agent Memory Systems",
		Content:   "# Agent Memory Systems\n\nThis document describes approaches to implementing memory systems for AI agents that emulate human memory patterns. Human memory consists of working memory, short-term memory, and long-term memory, each with different characteristics.",
		Tags:      []string{"memory", "agent", "cognitive"},
		CreatedBy: "system",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"importance": "high",
			"category":   "research",
		},
	}

	if err := store.CreateDocument(ctx, doc); err != nil {
		fmt.Printf("Error creating document: %v\n", err)
		return
	}
	fmt.Println("✓ Created document:", doc.ID)

	// Create a second document
	doc2 := &memory.Document{
		ID:        "doc-2",
		Title:     "Working Memory vs Short-Term Memory",
		Content:   "Working memory is active and manipulable, used for immediate reasoning. Short-term memory holds recent information but isn't actively manipulated. Both are limited in capacity and duration compared to long-term memory.",
		Tags:      []string{"memory", "cognitive", "working-memory", "short-term-memory"},
		CreatedBy: "system",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateDocument(ctx, doc2); err != nil {
		fmt.Printf("Error creating document: %v\n", err)
		return
	}
	fmt.Println("✓ Created document:", doc2.ID)

	// Create a third document
	doc3 := &memory.Document{
		ID:        "doc-3",
		Title:     "Long-Term Memory for Agents",
		Content:   "Long-term memory in agents should store important information indefinitely. It can be divided into episodic memory (experiences), semantic memory (facts), and procedural memory (skills). Efficient storage and retrieval are critical challenges.",
		Tags:      []string{"memory", "long-term-memory", "agent"},
		CreatedBy: "system",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateDocument(ctx, doc3); err != nil {
		fmt.Printf("Error creating document: %v\n", err)
		return
	}
	fmt.Println("✓ Created document:", doc3.ID)

	// Read a document
	readDoc, err := store.ReadDocument(ctx, "doc-1")
	if err != nil {
		fmt.Printf("Error reading document: %v\n", err)
		return
	}
	fmt.Println("✓ Read document:", readDoc.Title)

	// Update a document
	readDoc.Content += "\n\n## Working Memory\n\nWorking memory is the scratchpad of the mind, used for temporarily holding information for immediate processing and reasoning."
	readDoc.UpdatedAt = time.Now()
	readDoc.UpdatedBy = "system"

	if err := store.UpdateDocument(ctx, readDoc); err != nil {
		fmt.Printf("Error updating document: %v\n", err)
		return
	}
	fmt.Println("✓ Updated document:", readDoc.ID)

	// Search documents
	results, err := store.SearchDocuments(ctx, "working memory")
	if err != nil {
		fmt.Printf("Error searching documents: %v\n", err)
		return
	}
	fmt.Printf("✓ Found %d documents matching 'working memory':\n", len(results))
	for _, result := range results {
		fmt.Printf("  - %s\n", result.Title)
	}

	fmt.Println("\n=== Testing Memory Chunks & Vector Search ===")

	// Create memory chunks for better semantic search
	chunk1 := &memory.Chunk{
		ID:        "chunk-1",
		Content:   "Working memory is the cognitive system responsible for temporarily holding information for immediate mental processing and reasoning.",
		SourceID:  "doc-2",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"concept": "working memory",
			"section": "definition",
		},
	}

	if err := store.CreateChunk(ctx, chunk1); err != nil {
		fmt.Printf("Error creating chunk: %v\n", err)
		return
	}
	fmt.Println("✓ Created chunk:", chunk1.ID)

	chunk2 := &memory.Chunk{
		ID:        "chunk-2",
		Content:   "Short-term memory temporarily holds information for a brief period without manipulation, typically 20-30 seconds without rehearsal.",
		SourceID:  "doc-2",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"concept": "short-term memory",
			"section": "definition",
		},
	}

	if err := store.CreateChunk(ctx, chunk2); err != nil {
		fmt.Printf("Error creating chunk: %v\n", err)
		return
	}
	fmt.Println("✓ Created chunk:", chunk2.ID)

	chunk3 := &memory.Chunk{
		ID:        "chunk-3",
		Content:   "Long-term memory stores information for extended periods, potentially indefinitely, with practically unlimited capacity.",
		SourceID:  "doc-3",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"concept": "long-term memory",
			"section": "definition",
		},
	}

	if err := store.CreateChunk(ctx, chunk3); err != nil {
		fmt.Printf("Error creating chunk: %v\n", err)
		return
	}
	fmt.Println("✓ Created chunk:", chunk3.ID)

	chunk4 := &memory.Chunk{
		ID:        "chunk-4",
		Content:   "Episodic memory stores specific experiences and events, including their context, time, and emotional associations.",
		SourceID:  "doc-3",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"concept": "episodic memory",
			"section": "types",
		},
	}

	if err := store.CreateChunk(ctx, chunk4); err != nil {
		fmt.Printf("Error creating chunk: %v\n", err)
		return
	}
	fmt.Println("✓ Created chunk:", chunk4.ID)

	// Test vector search
	queries := []string{
		"working memory capacity",
		"how long does short-term memory last",
		"types of long-term memory",
		"differences between memory systems",
		"episodic memory definition",
	}

	for _, query := range queries {
		// Search chunks with this query
		fmt.Printf("\nQuery: '%s'\n", query)
		
		chunkResults, err := store.SearchChunks(ctx, query, 2)
		if err != nil {
			fmt.Printf("Error searching chunks: %v\n", err)
			continue
		}
		
		fmt.Printf("Top %d semantically similar chunks:\n", len(chunkResults))
		for i, chunk := range chunkResults {
			fmt.Printf("%d. [%s] %s\n", i+1, chunk.ID, chunk.Content)
		}
	}
}