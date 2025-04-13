package llm

// Response represents a response from an LLM
type Response struct {
    Content      string
    ToolCalls    []ToolCall
    Usage        Usage
    FinishReason string
}

// ToolCall represents a tool call from an LLM
type ToolCall struct {
    ToolName    string
    Arguments   map[string]interface{}
    Response    interface{}
}

// Usage represents token usage information
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// NewAnthropicLLM creates a new Anthropic LLM instance
func NewAnthropicLLM(model string, params map[string]interface{}) (Interface, error) {
    llm := &AnthropicLLM{
        Model:  model,
        Params: params,
    }
    return llm, nil
}

// NewOpenAILLM creates a new OpenAI LLM instance
func NewOpenAILLM(model string, params map[string]interface{}) (Interface, error) {
    llm := &OpenAILLM{
        Model:  model,
        Params: params,
    }
    return llm, nil
}
