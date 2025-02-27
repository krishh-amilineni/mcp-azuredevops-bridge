package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// Handler for adding attachment to work item
func handleAddWorkItemAttachment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	fileName := request.Params.Arguments["file_name"].(string)
	content := request.Params.Arguments["content"].(string)

	// Decode base64 content
	fileContent, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid base64 content: %v", err)), nil
	}

	// Create upload stream
	stream := bytes.NewReader(fileContent)

	// Upload attachment
	attachment, err := workItemClient.CreateAttachment(ctx, workitemtracking.CreateAttachmentArgs{
		UploadStream: stream,
		FileName:     &fileName,
		Project:      &config.Project,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to upload attachment: %v", err)), nil
	}

	// Add attachment reference to work item
	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:   &webapi.OperationValues.Add,
				Path: stringPtr("/relations/-"),
				Value: map[string]interface{}{
					"rel": "AttachedFile",
					"url": *attachment.Url,
					"attributes": map[string]interface{}{
						"name": fileName,
					},
				},
			},
		},
	}

	_, err = workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add attachment to work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added attachment '%s' to work item #%d", fileName, id)), nil
}

// Handler for getting work item attachments
func handleGetWorkItemAttachments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))

	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Expand:  &workitemtracking.WorkItemExpandValues.Relations,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	if workItem.Relations == nil {
		return mcp.NewToolResultText(fmt.Sprintf("No attachments found for work item #%d", id)), nil
	}

	var results []string
	for _, relation := range *workItem.Relations {
		if *relation.Rel == "AttachedFile" {
			name := (*relation.Attributes)["name"].(string)
			results = append(results, fmt.Sprintf("ID: %s\nName: %s\nURL: %s\n---",
				*relation.Url,
				name,
				*relation.Url))
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No attachments found for work item #%d", id)), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for removing attachment from work item
func handleRemoveWorkItemAttachment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	attachmentID := request.Params.Arguments["attachment_id"].(string)

	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Expand:  &workitemtracking.WorkItemExpandValues.Relations,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	if workItem.Relations == nil {
		return mcp.NewToolResultError("Work item has no attachments"), nil
	}

	// Find the attachment relation index
	var relationIndex int = -1
	for i, relation := range *workItem.Relations {
		if *relation.Rel == "AttachedFile" && strings.Contains(*relation.Url, attachmentID) {
			relationIndex = i
			break
		}
	}

	if relationIndex == -1 {
		return mcp.NewToolResultError("Attachment not found"), nil
	}

	// Remove the attachment relation
	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:   &webapi.OperationValues.Remove,
				Path: stringPtr(fmt.Sprintf("/relations/%d", relationIndex)),
			},
		},
	}

	_, err = workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove attachment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed attachment from work item #%d", id)), nil
}
