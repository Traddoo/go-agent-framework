package main

import (
	"context"
	"fmt"
	"time"

	"github.com/traddoo/go-agent-framework/pkg/memory"
	"github.com/traddoo/go-agent-framework/pkg/memory/cognitive"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create a memory store for long-term storage
	store := memory.NewInMemoryStore()
	if err := store.Initialize(ctx); err != nil {
		fmt.Printf("Error initializing memory store: %v\n", err)
		return
	}

	// Create a cognitive memory system for our agent
	cogMem := cognitive.NewCognitiveMemory(store, "agent-1")

	fmt.Println("=== Testing Cognitive Memory System ===")

	// Add some items to working memory (scratchpad)
	fmt.Println("\n--- Adding to Working Memory ---")
	item1, err := cogMem.AddToWorkingMemory(
		"The user wants me to build a web scraper that can extract product information from e-commerce sites.",
		0.8, // Important
		map[string]interface{}{
			"type": "task",
			"tags": []string{"web-scraper", "e-commerce"},
		},
	)
	if err != nil {
		fmt.Printf("Error adding to working memory: %v\n", err)
		return
	}
	fmt.Printf("✓ Added to working memory: %s (importance: %.1f)\n", item1.ID, item1.Importance)

	item2, err := cogMem.AddToWorkingMemory(
		"I'll need to use a library like BeautifulSoup or Selenium for this task.",
		0.6, // Moderately important
		map[string]interface{}{
			"type": "thought",
			"tags": []string{"libraries", "tools"},
		},
	)
	if err != nil {
		fmt.Printf("Error adding to working memory: %v\n", err)
		return
	}
	fmt.Printf("✓ Added to working memory: %s (importance: %.1f)\n", item2.ID, item2.Importance)

	item3, err := cogMem.AddToWorkingMemory(
		"The user mentioned they prefer Python over JavaScript for this project.",
		0.7, // Important
		map[string]interface{}{
			"type": "requirement",
			"tags": []string{"language", "preference"},
		},
	)
	if err != nil {
		fmt.Printf("Error adding to working memory: %v\n", err)
		return
	}
	fmt.Printf("✓ Added to working memory: %s (importance: %.1f)\n", item3.ID, item3.Importance)

	// Add more items to trigger consolidation to short-term memory
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("Additional thought %d about the web scraper project", i+1)
		importance := 0.3 + float64(i)*0.1 // Less important
		item, err := cogMem.AddToWorkingMemory(
			content,
			importance,
			map[string]interface{}{
				"type": "thought",
				"tags": []string{"planning"},
			},
		)
		if err != nil {
			fmt.Printf("Error adding to working memory: %v\n", err)
			return
		}
		fmt.Printf("✓ Added to working memory: %s (importance: %.1f)\n", item.ID, item.Importance)
	}

	// Check working memory state
	fmt.Println("\n--- Working Memory Contents ---")
	workingMem := cogMem.GetWorkingMemory()
	for i, item := range workingMem {
		fmt.Printf("%d. [%.1f] %s\n", i+1, item.Importance, item.Content)
	}

	// Check short-term memory state
	fmt.Println("\n--- Short-Term Memory Contents ---")
	shortTermMem := cogMem.GetShortTermMemory()
	for i, item := range shortTermMem {
		fmt.Printf("%d. [%.1f] %s\n", i+1, item.Importance, item.Content)
	}

	// Recall information
	fmt.Println("\n--- Testing Memory Recall ---")
	queries := []string{
		"web scraper",
		"Python",
		"libraries",
		"requirements",
	}

	for _, query := range queries {
		fmt.Printf("\nRecalling '%s' from all memory systems:\n", query)
		results, err := cogMem.Recall(ctx, query, []cognitive.MemoryType{
			cognitive.WorkingMemory,
			cognitive.ShortTermMemory,
			cognitive.LongTermMemory,
		})
		if err != nil {
			fmt.Printf("Error recalling memories: %v\n", err)
			continue
		}

		fmt.Printf("Found %d memories:\n", len(results))
		for i, result := range results {
			switch mem := result.(type) {
			case *cognitive.MemoryItem:
				fmt.Printf("%d. [%s] %s\n", i+1, mem.Type, mem.Content)
			case *memory.Chunk:
				fmt.Printf("%d. [long-term] %s\n", i+1, mem.Content)
			}
		}
	}

	// Reflect on important memories to strengthen them
	fmt.Println("\n--- Testing Memory Reflection ---")
	if len(shortTermMem) > 0 {
		itemToReflect := shortTermMem[0].ID
		fmt.Printf("Reflecting on memory: %s\n", itemToReflect)
		err := cogMem.Reflect(ctx, []string{itemToReflect}, 0.3)
		if err != nil {
			fmt.Printf("Error reflecting on memory: %v\n", err)
		}
	}

	// Memory maintenance
	fmt.Println("\n--- Running Memory Maintenance ---")
	err = cogMem.MaintainMemories(ctx)
	if err != nil {
		fmt.Printf("Error maintaining memories: %v\n", err)
	}

	// Check memory state after maintenance
	fmt.Println("\n--- Memory State After Maintenance ---")
	fmt.Println("Working Memory:")
	for i, item := range cogMem.GetWorkingMemory() {
		fmt.Printf("%d. [%.1f] %s\n", i+1, item.Importance, item.Content)
	}

	fmt.Println("\nShort-Term Memory:")
	for i, item := range cogMem.GetShortTermMemory() {
		fmt.Printf("%d. [%.1f] %s\n", i+1, item.Importance, item.Content)
	}

	// Forgetting
	fmt.Println("\n--- Testing Memory Forgetting ---")
	if len(workingMem) > 0 {
		itemToForget := workingMem[0].ID
		fmt.Printf("Forgetting memory: %s\n", itemToForget)
		err := cogMem.Forget(ctx, []string{itemToForget})
		if err != nil {
			fmt.Printf("Error forgetting memory: %v\n", err)
		}
	}

	// Final memory state
	fmt.Println("\n--- Final Memory State ---")
	fmt.Println("Working Memory:")
	for i, item := range cogMem.GetWorkingMemory() {
		fmt.Printf("%d. [%.1f] %s\n", i+1, item.Importance, item.Content)
	}
}