package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// BraveSearchResult represents a single search result from Brave Search API
type BraveSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// BraveWebSearchResponse represents the response structure from Brave Search API
type BraveWebSearchResponse struct {
	Type  string `json:"type"`
	Query struct {
		Original          string `json:"original"`
		ShowStrictWarning bool   `json:"show_strict_warning"`
		IsNavigational    bool   `json:"is_navigational"`
		IsSafeSearchActive bool  `json:"is_safe_search_active"`
	} `json:"query"`
	Web struct {
		Results              []BraveSearchResult `json:"results"`
		MoreResultsAvailable bool               `json:"more_results_available"`
	} `json:"web"`
}

// BraveSearchTool implements a tool for web searches using Brave Search API
type BraveSearchTool struct {
	apiKey   string
	endpoint string
}

// NewBraveSearchTool creates a new Brave Search tool instance
func NewBraveSearchTool(apiKey string) *BraveSearchTool {
	// If API key not provided, try to get from environment
	if apiKey == "" {
		apiKey = os.Getenv("BRAVE_API_KEY")
	}

	return &BraveSearchTool{
		apiKey:   apiKey,
		endpoint: "https://api.search.brave.com/res/v1/web/search",
	}
}

// Name returns the name of the tool
func (t *BraveSearchTool) Name() string {
	return "brave_search"
}

// Description returns a description of the tool
func (t *BraveSearchTool) Description() string {
	return "Search the web for information on a specific query using Brave Search"
}

// Parameters returns the parameters for the tool
func (t *BraveSearchTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query",
			},
			"num_results": map[string]interface{}{
				"type":        "integer",
				"default":     5,
				"description": "Number of results to return",
			},
			"country": map[string]interface{}{
				"type":        "string",
				"description": "Country code for localized results (e.g., \"us\", \"uk\", \"fr\")",
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool
func (t *BraveSearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract arguments
	query, ok := args["query"].(string)
	if !ok {
		return nil, errors.New("query parameter is required")
	}

	// Get optional parameters
	numResults := 5
	if numResultsArg, ok := args["num_results"].(float64); ok {
		numResults = int(numResultsArg)
	}

	country := ""
	if countryArg, ok := args["country"].(string); ok {
		country = countryArg
	}

	// Validate API key
	if t.apiKey == "" {
		// If API key is missing, return mock results instead of failing
		return t.getMockResults(query, numResults), nil
	}

	// Build search URL with parameters
	searchURL, err := url.Parse(t.endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	q := searchURL.Query()
	q.Add("q", query)
	if country != "" {
		q.Add("country", country)
	}
	searchURL.RawQuery = q.Encode()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("X-Subscription-Token", t.apiKey)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing search request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		// If rate limited, return mock results
		if resp.StatusCode == http.StatusTooManyRequests {
			return t.getMockResults(query, numResults), nil
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var searchResponse BraveWebSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Format and limit results
	results := make([]map[string]string, 0)
	for i, result := range searchResponse.Web.Results {
		if i >= numResults {
			break
		}
		results = append(results, map[string]string{
			"title":   result.Title,
			"snippet": result.Description,
			"url":     result.URL,
		})
	}

	return map[string]interface{}{
		"query":   query,
		"results": results,
	}, nil
}

// getMockResults returns mock search results when API key is missing or rate limited
func (t *BraveSearchTool) getMockResults(query string, numResults int) map[string]interface{} {
	mockResults := []map[string]string{
		{
			"title":   fmt.Sprintf("Mock result for \"%s\"", query),
			"snippet": fmt.Sprintf("This is a mock search result for \"%s\" because the Brave Search API key is missing or rate limited.", query),
			"url":     fmt.Sprintf("https://example.com/mock?q=%s", url.QueryEscape(query)),
		},
		{
			"title":   fmt.Sprintf("Information about %s - Wikipedia", query),
			"snippet": fmt.Sprintf("This would be a summary of information about %s from a reputable encyclopedia source.", query),
			"url":     fmt.Sprintf("https://example.com/wiki/%s", url.QueryEscape(query)),
		},
		{
			"title":   fmt.Sprintf("Latest news on %s", query),
			"snippet": fmt.Sprintf("Recent developments and news related to %s from various sources.", query),
			"url":     fmt.Sprintf("https://example.com/news/%s", url.QueryEscape(query)),
		},
		{
			"title":   fmt.Sprintf("Research papers about %s", query),
			"snippet": fmt.Sprintf("Academic articles and research publications discussing %s and related topics.", query),
			"url":     fmt.Sprintf("https://example.com/research/%s", url.QueryEscape(query)),
		},
		{
			"title":   fmt.Sprintf("How to learn more about %s", query),
			"snippet": fmt.Sprintf("Resources, tutorials, and guides for learning about %s in depth.", query),
			"url":     fmt.Sprintf("https://example.com/learn/%s", url.QueryEscape(query)),
		},
	}

	// Limit results to requested number
	if numResults < len(mockResults) {
		mockResults = mockResults[:numResults]
	}

	return map[string]interface{}{
		"query":   query,
		"results": mockResults,
		"note":    "These are mock results. To get real results, provide a Brave Search API key.",
	}
}