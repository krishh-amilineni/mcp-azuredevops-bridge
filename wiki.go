package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/wiki"
)

func addWikiTools(s *server.MCPServer) {
	// Wiki Page Management
	manageWikiTool := mcp.NewTool("manage_wiki_page",
		mcp.WithDescription("Create or update a wiki page"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path of the wiki page"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Content of the wiki page in markdown format"),
		),
	)
	s.AddTool(manageWikiTool, handleManageWikiPage)

	// Get Wiki Page
	getWikiTool := mcp.NewTool("get_wiki_page",
		mcp.WithDescription("Get content of a wiki page"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path of the wiki page to retrieve"),
		),
		mcp.WithBoolean("include_children",
			mcp.Description("Whether to include child pages"),
		),
	)
	s.AddTool(getWikiTool, handleGetWikiPage)

	// List Wiki Pages
	listWikiTool := mcp.NewTool("list_wiki_pages",
		mcp.WithDescription("List wiki pages in a directory"),
		mcp.WithString("path",
			mcp.Description("Path to list pages from (optional)"),
		),
		mcp.WithBoolean("recursive",
			mcp.Description("Whether to list pages recursively"),
		),
	)
	s.AddTool(listWikiTool, handleListWikiPages)

	// Search Wiki
	searchWikiTool := mcp.NewTool("search_wiki",
		mcp.WithDescription("Search wiki pages"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithString("path",
			mcp.Description("Path to limit search to (optional)"),
		),
	)
	s.AddTool(searchWikiTool, handleSearchWiki)

	// Get Available Wikis
	getWikisTool := mcp.NewTool("get_available_wikis",
		mcp.WithDescription("Get information about available wikis"),
	)
	s.AddTool(getWikisTool, handleGetWikis)
}

func handleManageWikiPage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := request.Params.Arguments["path"].(string)
	content := request.Params.Arguments["content"].(string)
	// Note: Comments are not supported by the Azure DevOps Wiki API
	_, _ = request.Params.Arguments["comment"].(string)

	// Get all available wikis for the project
	wikis, err := getWikisForProject(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wikis: %v", err)), nil
	}

	if len(wikis) == 0 {
		return mcp.NewToolResultError("No wikis found for this project"), nil
	}

	// Use the first wiki by default, or try to match by project name
	wikiId := *wikis[0].Id
	for _, wiki := range wikis {
		if strings.Contains(*wiki.Name, config.Project) {
			wikiId = *wiki.Id
			break
		}
	}

	// Convert wiki ID to the format expected by the API
	wikiIdentifier := fmt.Sprintf("%s", wikiId)

	_, err = wikiClient.CreateOrUpdatePage(ctx, wiki.CreateOrUpdatePageArgs{
		WikiIdentifier: &wikiIdentifier,
		Path:           &path,
		Project:        &config.Project,
		Parameters: &wiki.WikiPageCreateOrUpdateParameters{
			Content: &content,
		},
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to manage wiki page: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully managed wiki page: %s", path)), nil
}

func handleGetWikiPage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := request.Params.Arguments["path"].(string)
	includeChildren, _ := request.Params.Arguments["include_children"].(bool)

	// Ensure path starts with a forward slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	log.Printf("Wiki page path: %s", path)

	recursionLevel := "none"
	if includeChildren {
		recursionLevel = "oneLevel"
	}

	// Get all available wikis for the project
	wikis, err := getWikisForProject(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wikis: %v", err)), nil
	}

	log.Printf("Found %d wikis for project", len(wikis))
	for i, wiki := range wikis {
		log.Printf("Wiki %d: %s (ID: %s)", i+1, *wiki.Name, *wiki.Id)
	}

	if len(wikis) == 0 {
		return mcp.NewToolResultError("No wikis found for this project"), nil
	}

	// Use the first wiki by default
	wikiId := *wikis[0].Id
	
	// Try to find a wiki with a name that matches or contains the project name
	projectName := strings.Replace(config.Project, " ", "", -1)
	projectName = strings.ToLower(projectName)
	
	for _, wiki := range wikis {
		wikiName := strings.ToLower(*wiki.Name)
		if strings.Contains(wikiName, projectName) || strings.Contains(wikiName, "documentation") {
			wikiId = *wiki.Id
			log.Printf("Selected wiki: %s (ID: %s)", *wiki.Name, wikiId)
			break
		}
	}

	// Build the URL with query parameters
	baseURL := fmt.Sprintf("%s/%s/_apis/wiki/wikis/%s/pages",
		config.OrganizationURL,
		url.PathEscape(config.Project),
		wikiId)

	queryParams := url.Values{}
	queryParams.Add("path", path)
	queryParams.Add("recursionLevel", recursionLevel)
	queryParams.Add("includeContent", "true")
	queryParams.Add("api-version", "7.2-preview")

	fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())
	log.Printf("Requesting wiki page from URL: %s", fullURL)

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Add authentication
	req.SetBasicAuth("", config.PersonalAccessToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wiki page: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response body: %v", err)), nil
	}

	if resp.StatusCode != http.StatusOK {
		// Log more details about the error
		log.Printf("Wiki API Error - Status: %d, Response: %s", resp.StatusCode, string(responseBody))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wiki page. Status: %d", resp.StatusCode)), nil
	}

	// Parse response
	var wikiResponse struct {
		Content  string `json:"content"`
		SubPages []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"subPages"`
	}

	log.Printf("Wiki API Response: %s", string(responseBody))
	
	if err := json.Unmarshal(responseBody, &wikiResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	// Format result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== %s ===\n\n", path))
	result.WriteString(wikiResponse.Content)

	if includeChildren && len(wikiResponse.SubPages) > 0 {
		result.WriteString("\n\nSub-pages:\n")
		for _, subPage := range wikiResponse.SubPages {
			result.WriteString(fmt.Sprintf("\n=== %s ===\n", subPage.Path))
			result.WriteString(subPage.Content)
			result.WriteString("\n")
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleListWikiPages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := request.Params.Arguments["path"].(string)
	recursive, _ := request.Params.Arguments["recursive"].(bool)

	recursionLevel := "oneLevel"
	if recursive {
		recursionLevel = "full"
	}

	// Get all available wikis for the project
	wikis, err := getWikisForProject(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wikis: %v", err)), nil
	}

	if len(wikis) == 0 {
		return mcp.NewToolResultError("No wikis found for this project"), nil
	}

	// Use the first wiki by default, or try to match by project name
	wikiId := *wikis[0].Id
	for _, wiki := range wikis {
		if strings.Contains(*wiki.Name, config.Project) {
			wikiId = *wiki.Id
			break
		}
	}

	// Build the URL with query parameters
	baseURL := fmt.Sprintf("%s/%s/_apis/wiki/wikis/%s/pages",
		config.OrganizationURL,
		url.PathEscape(config.Project),
		wikiId)

	queryParams := url.Values{}
	if path != "" {
		queryParams.Add("path", path)
	}
	queryParams.Add("recursionLevel", recursionLevel)
	queryParams.Add("api-version", "7.2-preview")

	fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Add authentication
	req.SetBasicAuth("", config.PersonalAccessToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list wiki pages: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response body: %v", err)), nil
	}

	if resp.StatusCode != http.StatusOK {
		// Log error details
		log.Printf("Wiki API Error - Status: %d, Response: %s", resp.StatusCode, string(responseBody))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list wiki pages. Status: %d", resp.StatusCode)), nil
	}

	// Parse response
	var listResponse struct {
		Value []struct {
			Path       string `json:"path"`
			RemotePath string `json:"remotePath"`
			IsFolder   bool   `json:"isFolder"`
		} `json:"value"`
	}

	log.Printf("Wiki API Response: %s", string(responseBody))
	
	if err := json.Unmarshal(responseBody, &listResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	// Format result
	var result strings.Builder
	var locationText string
	if path != "" {
		locationText = " in " + path
	}
	result.WriteString(fmt.Sprintf("Wiki pages%s:\n\n", locationText))

	for _, item := range listResponse.Value {
		prefix := "📄 "
		if item.IsFolder {
			prefix = "📁 "
		}
		result.WriteString(fmt.Sprintf("%s%s\n", prefix, item.Path))
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleSearchWiki(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.Params.Arguments["query"].(string)
	path, hasPath := request.Params.Arguments["path"].(string)

	// Get all available wikis for the project
	wikis, err := getWikisForProject(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wikis: %v", err)), nil
	}

	if len(wikis) == 0 {
		return mcp.NewToolResultError("No wikis found for this project"), nil
	}

	// Use the first wiki by default, or try to match by project name
	wikiId := *wikis[0].Id
	for _, wiki := range wikis {
		if strings.Contains(*wiki.Name, config.Project) {
			wikiId = *wiki.Id
			break
		}
	}

	// First, get all pages (potentially under the specified path)
	baseURL := fmt.Sprintf("%s/%s/_apis/wiki/wikis/%s/pages",
		config.OrganizationURL,
		url.PathEscape(config.Project),
		wikiId)

	queryParams := url.Values{}
	queryParams.Add("recursionLevel", "full")
	if hasPath {
		queryParams.Add("path", path)
	}
	queryParams.Add("includeContent", "true")
	queryParams.Add("api-version", "7.2-preview")

	fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Add authentication
	req.SetBasicAuth("", config.PersonalAccessToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search wiki: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response body: %v", err)), nil
	}

	if resp.StatusCode != http.StatusOK {
		// Log error details
		log.Printf("Wiki API Error - Status: %d, Response: %s", resp.StatusCode, string(responseBody))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search wiki. Status: %d", resp.StatusCode)), nil
	}

	// Parse response
	var searchResponse struct {
		Count int `json:"count"`
		Results []struct {
			FileName    string `json:"fileName"`
			Path        string `json:"path"`
			MatchCount  int    `json:"hitCount"`
			Repository  struct {
				ID string `json:"id"`
			} `json:"repository"`
			Hits []struct {
				Content    string `json:"content"`
				LineNumber int    `json:"startLine"`
			} `json:"hits"`
		} `json:"results"`
	}

	log.Printf("Wiki API Search Response: %s", string(responseBody))
	
	if err := json.Unmarshal(responseBody, &searchResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	// Search through the pages
	var results []string
	queryLower := strings.ToLower(query)
	for _, page := range searchResponse.Results {
		if strings.Contains(strings.ToLower(page.FileName), queryLower) {
			// Extract a snippet of context around the match
			contentLower := strings.ToLower(page.FileName)
			index := strings.Index(contentLower, queryLower)
			start := 0
			if index > 100 {
				start = index - 100
			}
			end := len(page.FileName)
			if index+len(query)+100 < len(page.FileName) {
				end = index + len(query) + 100
			}

			snippet := page.FileName[start:end]
			if start > 0 {
				snippet = "..." + snippet
			}
			if end < len(page.FileName) {
				snippet = snippet + "..."
			}

			results = append(results, fmt.Sprintf("Page: %s\nMatch: %s\n---\n", page.Path, snippet))
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No matches found for '%s'", query)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d matches:\n\n%s", len(results), strings.Join(results, "\n"))), nil
}

func handleGetWikis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	wikis, err := getWikisForProject(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get wikis: %v", err)), nil
	}

	if len(wikis) == 0 {
		return mcp.NewToolResultError("No wikis found for this project"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d wikis for project %s:\n\n", len(wikis), config.Project))

	for i, wiki := range wikis {
		result.WriteString(fmt.Sprintf("%d. Wiki Name: %s\n   Wiki ID: %s\n\n",
			i+1, *wiki.Name, *wiki.Id))
	}

	return mcp.NewToolResultText(result.String()), nil
}

func getWikisForProject(ctx context.Context) ([]*wiki.Wiki, error) {
	// Create request
	wikiApiUrl := fmt.Sprintf("%s/%s/_apis/wiki/wikis?api-version=7.2-preview", 
		config.OrganizationURL,
		url.PathEscape(config.Project))
	log.Printf("Getting wikis from URL: %s", wikiApiUrl)
	
	req, err := http.NewRequest("GET", wikiApiUrl, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication
	req.SetBasicAuth("", config.PersonalAccessToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Wiki API Status Code: %d", resp.StatusCode)
	
	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response: %s", string(bodyBytes))
		return nil, fmt.Errorf("Failed to get wikis. Status: %d", resp.StatusCode)
	}

	// Parse response
	var wikisResponse struct {
		Value []*wiki.Wiki `json:"value"`
	}
	
	log.Printf("Wiki API Response: %s", string(bodyBytes))
	
	// Unmarshal JSON directly from the bytes
	if err := json.Unmarshal(bodyBytes, &wikisResponse); err != nil {
		return nil, fmt.Errorf("Failed to parse wikis response: %v", err)
	}

	log.Printf("Found %d wikis in total", len(wikisResponse.Value))
	
	// For now, return all wikis since we don't have a reliable way to filter
	// If needed, we can add more specific filtering later
	if len(wikisResponse.Value) > 0 {
		log.Printf("First wiki: Name=%s, ID=%s", *wikisResponse.Value[0].Name, *wikisResponse.Value[0].Id)
	}
	
	return wikisResponse.Value, nil
}
