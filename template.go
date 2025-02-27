package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// Handler for getting work item templates
func handleGetWorkItemTemplates(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workItemType := request.Params.Arguments["type"].(string)

	templates, err := workItemClient.GetTemplates(ctx, workitemtracking.GetTemplatesArgs{
		Project:          &config.Project,
		Team:             nil, // Get templates for entire project
		Workitemtypename: &workItemType,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get templates: %v", err)), nil
	}

	var results []string
	for _, template := range *templates {
		results = append(results, fmt.Sprintf("Template ID: %s\nName: %s\nDescription: %s\n---",
			*template.Id,
			*template.Name,
			*template.Description))
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No templates found for type: %s", workItemType)), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for creating work item from template
func handleCreateFromTemplate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templateID := request.Params.Arguments["template_id"].(string)
	fieldValuesJSON := request.Params.Arguments["field_values"].(string)

	var fieldValues map[string]interface{}
	if err := json.Unmarshal([]byte(fieldValuesJSON), &fieldValues); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid field values JSON: %v", err)), nil
	}

	// Convert template ID to UUID
	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid template ID format: %v", err)), nil
	}

	// Get template
	template, err := workItemClient.GetTemplate(ctx, workitemtracking.GetTemplateArgs{
		Project:    &config.Project,
		Team:       nil,
		TemplateId: &templateUUID,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get template: %v", err)), nil
	}

	// Create work item from template
	createArgs := workitemtracking.CreateWorkItemArgs{
		Type:    template.WorkItemTypeName,
		Project: &config.Project,
	}

	// Add template fields
	var operations []webapi.JsonPatchOperation
	for field, value := range *template.Fields {
		operations = append(operations, webapi.JsonPatchOperation{
			Op:    &webapi.OperationValues.Add,
			Path:  stringPtr("/fields/" + field),
			Value: value,
		})
	}

	// Override with provided field values
	for field, value := range fieldValues {
		operations = append(operations, webapi.JsonPatchOperation{
			Op:    &webapi.OperationValues.Add,
			Path:  stringPtr("/fields/" + field),
			Value: value,
		})
	}

	createArgs.Document = &operations

	workItem, err := workItemClient.CreateWorkItem(ctx, createArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create work item from template: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created work item #%d from template", *workItem.Id)), nil
}
