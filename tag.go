package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// Handler for managing work item tags
func handleManageWorkItemTags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	operation := request.Params.Arguments["operation"].(string)
	tagsStr := request.Params.Arguments["tags"].(string)
	tags := strings.Split(tagsStr, ",")

	// Get current work item to get existing tags
	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	fields := *workItem.Fields
	var currentTags []string
	if tags, ok := fields["System.Tags"].(string); ok && tags != "" {
		currentTags = strings.Split(tags, "; ")
	}

	var newTags []string
	switch operation {
	case "add":
		// Add new tags while avoiding duplicates
		tagMap := make(map[string]bool)
		for _, tag := range currentTags {
			tagMap[strings.TrimSpace(tag)] = true
		}
		for _, tag := range tags {
			tagMap[strings.TrimSpace(tag)] = true
		}
		for tag := range tagMap {
			newTags = append(newTags, tag)
		}
	case "remove":
		// Remove specified tags
		tagMap := make(map[string]bool)
		for _, tag := range tags {
			tagMap[strings.TrimSpace(tag)] = true
		}
		for _, tag := range currentTags {
			if !tagMap[strings.TrimSpace(tag)] {
				newTags = append(newTags, tag)
			}
		}
	}

	// Update work item with new tags
	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:    &webapi.OperationValues.Replace,
				Path:  stringPtr("/fields/System.Tags"),
				Value: strings.Join(newTags, "; "),
			},
		},
	}

	_, err = workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update tags: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully %sd tags for work item #%d", operation, id)), nil
}

// Handler for getting work item tags
func handleGetWorkItemTags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))

	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	fields := *workItem.Fields
	if tags, ok := fields["System.Tags"].(string); ok && tags != "" {
		return mcp.NewToolResultText(fmt.Sprintf("Tags for work item #%d:\n%s", id, tags)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("No tags found for work item #%d", id)), nil
}
