package tools

import (
    "context"
    "github.com/traddoo/go-agent-framework/pkg/memory"
)

// DocumentWriteTool is a tool for writing documents to memory
type DocumentWriteTool struct {
    MemoryStore memory.Interface
}

func (t *DocumentWriteTool) Name() string {
    return "document_write"
}

func (t *DocumentWriteTool) Description() string {
    return "Write or update a document in the memory store"
}

func (t *DocumentWriteTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    documentID := args["document_id"].(string)
    content := args["content"].(string)
    
    // Read existing document
    doc, err := t.MemoryStore.ReadDocument(ctx, documentID)
    if err != nil {
        return nil, ToolError{
            Code:    "document_read_error",
            Message: "Failed to read document: " + err.Error(),
        }
    }
    
    // Update content
    doc.Content = content
    
    // Update optional fields
    if title, ok := args["title"].(string); ok {
        doc.Title = title
    }
    
    // Save document
    if err := t.MemoryStore.UpdateDocument(ctx, doc); err != nil {
        return nil, ToolError{
            Code:    "document_write_error",
            Message: "Failed to update document: " + err.Error(),
        }
    }
    
    return map[string]interface{}{
        "status": "success",
        "document_id": documentID,
    }, nil
}