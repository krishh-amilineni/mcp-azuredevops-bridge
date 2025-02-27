package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleGetCurrentSprint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	team, _ := request.Params.Arguments["team"].(string)
	if team == "" {
		team = config.Project + " Team" // Default team name
	}

	// Build the URL for the current iteration
	baseURL := fmt.Sprintf("%s/%s/_apis/work/teamsettings/iterations",
		config.OrganizationURL,
		config.Project)

	queryParams := url.Values{}
	queryParams.Add("$timeframe", "current")
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
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get current sprint: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get current sprint. Status: %d", resp.StatusCode)), nil
	}

	// Parse response
	var sprintResponse struct {
		Value []struct {
			Name      string    `json:"name"`
			StartDate time.Time `json:"startDate"`
			EndDate   time.Time `json:"finishDate"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sprintResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	if len(sprintResponse.Value) == 0 {
		return mcp.NewToolResultText("No active sprint found"), nil
	}

	sprint := sprintResponse.Value[0]
	result := fmt.Sprintf("Current Sprint: %s\nStart Date: %s\nEnd Date: %s",
		sprint.Name,
		sprint.StartDate.Format("2006-01-02"),
		sprint.EndDate.Format("2006-01-02"))

	return mcp.NewToolResultText(result), nil
}

func handleGetSprints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	team, _ := request.Params.Arguments["team"].(string)
	includeCompleted, _ := request.Params.Arguments["include_completed"].(bool)
	if team == "" {
		team = config.Project + " Team"
	}

	// Build the URL for iterations
	baseURL := fmt.Sprintf("%s/%s/_apis/work/teamsettings/iterations",
		config.OrganizationURL,
		config.Project)

	queryParams := url.Values{}
	if !includeCompleted {
		queryParams.Add("$timeframe", "current,future")
	}
	queryParams.Add("api-version", "7.2-preview")

	fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	req.SetBasicAuth("", config.PersonalAccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get sprints: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get sprints. Status: %d", resp.StatusCode)), nil
	}

	var sprintResponse struct {
		Value []struct {
			Name      string    `json:"name"`
			StartDate time.Time `json:"startDate"`
			EndDate   time.Time `json:"finishDate"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sprintResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	var results []string
	for _, sprint := range sprintResponse.Value {
		results = append(results, fmt.Sprintf("Sprint: %s\nStart: %s\nEnd: %s\n---",
			sprint.Name,
			sprint.StartDate.Format("2006-01-02"),
			sprint.EndDate.Format("2006-01-02")))
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No sprints found"), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}
