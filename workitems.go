package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

func addWorkItemTools(s *server.MCPServer) {
	// Add WIQL Query Format Prompt
	s.AddPrompt(mcp.NewPrompt("wiql_query_format",
		mcp.WithPromptDescription("Helper for formatting WIQL queries for common scenarios"),
		mcp.WithArgument("query_type",
			mcp.ArgumentDescription("Type of query to format (current_sprint, assigned_to_me, etc)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("additional_fields",
			mcp.ArgumentDescription("Additional fields to include in the SELECT clause"),
		),
	), handleWiqlQueryFormatPrompt)

	// Create Work Item
	createWorkItemTool := mcp.NewTool("create_work_item",
		mcp.WithDescription("Create a new work item in Azure DevOps"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Type of work item (Epic, Feature, User Story, Task, Bug)"),
			mcp.Enum("Epic", "Feature", "User Story", "Task", "Bug"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Title of the work item"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of the work item"),
		),
		mcp.WithString("priority",
			mcp.Description("Priority of the work item (1-4)"),
			mcp.Enum("1", "2", "3", "4"),
		),
	)

	s.AddTool(createWorkItemTool, handleCreateWorkItem)

	// Update Work Item
	updateWorkItemTool := mcp.NewTool("update_work_item",
		mcp.WithDescription("Update an existing work item in Azure DevOps"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item to update"),
		),
		mcp.WithString("field",
			mcp.Required(),
			mcp.Description("Field to update (Title, Description, State, Priority)"),
			mcp.Enum("Title", "Description", "State", "Priority"),
		),
		mcp.WithString("value",
			mcp.Required(),
			mcp.Description("New value for the field"),
		),
	)

	s.AddTool(updateWorkItemTool, handleUpdateWorkItem)

	// Query Work Items
	queryWorkItemsTool := mcp.NewTool("query_work_items",
		mcp.WithDescription("Query work items using WIQL"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("WIQL query string"),
		),
	)

	s.AddTool(queryWorkItemsTool, handleQueryWorkItems)

	// Get Work Item Details
	getWorkItemTool := mcp.NewTool("get_work_item_details",
		mcp.WithDescription("Get detailed information about work items"),
		mcp.WithString("ids",
			mcp.Required(),
			mcp.Description("Comma-separated list of work item IDs"),
		),
	)
	s.AddTool(getWorkItemTool, handleGetWorkItemDetails)

	// Manage Work Item Relations
	manageRelationsTool := mcp.NewTool("manage_work_item_relations",
		mcp.WithDescription("Manage relationships between work items"),
		mcp.WithNumber("source_id",
			mcp.Required(),
			mcp.Description("ID of the source work item"),
		),
		mcp.WithNumber("target_id",
			mcp.Required(),
			mcp.Description("ID of the target work item"),
		),
		mcp.WithString("relation_type",
			mcp.Required(),
			mcp.Description("Type of relationship to manage"),
			mcp.Enum("parent", "child", "related"),
		),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation to perform"),
			mcp.Enum("add", "remove"),
		),
	)
	s.AddTool(manageRelationsTool, handleManageWorkItemRelations)

	// Get Related Work Items
	getRelatedItemsTool := mcp.NewTool("get_related_work_items",
		mcp.WithDescription("Get related work items"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item to get relations for"),
		),
		mcp.WithString("relation_type",
			mcp.Required(),
			mcp.Description("Type of relationships to get"),
			mcp.Enum("parent", "children", "related", "all"),
		),
	)
	s.AddTool(getRelatedItemsTool, handleGetRelatedWorkItems)

	// Comment Management Tool (as Discussion)
	addCommentTool := mcp.NewTool("add_work_item_comment",
		mcp.WithDescription("Add a comment to a work item as a discussion"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("Comment text"),
		),
	)
	s.AddTool(addCommentTool, handleAddWorkItemComment)

	getCommentsTool := mcp.NewTool("get_work_item_comments",
		mcp.WithDescription("Get comments for a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
	)
	s.AddTool(getCommentsTool, handleGetWorkItemComments)

	// Field Management Tool
	getFieldsTool := mcp.NewTool("get_work_item_fields",
		mcp.WithDescription("Get available work item fields and their current values"),
		mcp.WithNumber("work_item_id",
			mcp.Required(),
			mcp.Description("ID of the work item to examine fields from"),
		),
		mcp.WithString("field_name",
			mcp.Description("Optional field name to filter (case-insensitive partial match)"),
		),
	)
	s.AddTool(getFieldsTool, handleGetWorkItemFields)

	// Batch Operations Tools
	batchCreateTool := mcp.NewTool("batch_create_work_items",
		mcp.WithDescription("Create multiple work items in a single operation"),
		mcp.WithString("items",
			mcp.Required(),
			mcp.Description("JSON array of work items to create, each containing type, title, and description"),
		),
	)
	s.AddTool(batchCreateTool, handleBatchCreateWorkItems)

	batchUpdateTool := mcp.NewTool("batch_update_work_items",
		mcp.WithDescription("Update multiple work items in a single operation"),
		mcp.WithString("updates",
			mcp.Required(),
			mcp.Description("JSON array of updates, each containing id, field, and value"),
		),
	)
	s.AddTool(batchUpdateTool, handleBatchUpdateWorkItems)

	// Tag Management Tools
	manageTags := mcp.NewTool("manage_work_item_tags",
		mcp.WithDescription("Add or remove tags from a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation to perform"),
			mcp.Enum("add", "remove"),
		),
		mcp.WithString("tags",
			mcp.Required(),
			mcp.Description("Comma-separated list of tags"),
		),
	)
	s.AddTool(manageTags, handleManageWorkItemTags)

	getTagsTool := mcp.NewTool("get_work_item_tags",
		mcp.WithDescription("Get tags for a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
	)
	s.AddTool(getTagsTool, handleGetWorkItemTags)

	// Work Item Template Tools
	getTemplatesTool := mcp.NewTool("get_work_item_templates",
		mcp.WithDescription("Get available work item templates"),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Type of work item to get templates for"),
			mcp.Enum("Epic", "Feature", "User Story", "Task", "Bug"),
		),
	)
	s.AddTool(getTemplatesTool, handleGetWorkItemTemplates)

	createFromTemplateTool := mcp.NewTool("create_from_template",
		mcp.WithDescription("Create a work item from a template"),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to use"),
		),
		mcp.WithString("field_values",
			mcp.Required(),
			mcp.Description("JSON object of field values to override template defaults"),
		),
	)
	s.AddTool(createFromTemplateTool, handleCreateFromTemplate)

	// Attachment Management Tools
	addAttachmentTool := mcp.NewTool("add_work_item_attachment",
		mcp.WithDescription("Add an attachment to a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
		mcp.WithString("file_name",
			mcp.Required(),
			mcp.Description("Name of the file to attach"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Base64 encoded content of the file"),
		),
	)
	s.AddTool(addAttachmentTool, handleAddWorkItemAttachment)

	getAttachmentsTool := mcp.NewTool("get_work_item_attachments",
		mcp.WithDescription("Get attachments for a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
	)
	s.AddTool(getAttachmentsTool, handleGetWorkItemAttachments)

	removeAttachmentTool := mcp.NewTool("remove_work_item_attachment",
		mcp.WithDescription("Remove an attachment from a work item"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the work item"),
		),
		mcp.WithString("attachment_id",
			mcp.Required(),
			mcp.Description("ID of the attachment to remove"),
		),
	)
	s.AddTool(removeAttachmentTool, handleRemoveWorkItemAttachment)

	// Sprint Management Tools
	getCurrentSprintTool := mcp.NewTool("get_current_sprint",
		mcp.WithDescription("Get details about the current sprint"),
		mcp.WithString("team",
			mcp.Description("Team name (optional, defaults to project's default team)"),
		),
	)
	s.AddTool(getCurrentSprintTool, handleGetCurrentSprint)

	getSprintsTool := mcp.NewTool("get_sprints",
		mcp.WithDescription("Get list of sprints"),
		mcp.WithString("team",
			mcp.Description("Team name (optional, defaults to project's default team)"),
		),
		mcp.WithBoolean("include_completed",
			mcp.Description("Whether to include completed sprints"),
		),
	)
	s.AddTool(getSprintsTool, handleGetSprints)

	// Add a new prompt for work item descriptions
	s.AddPrompt(mcp.NewPrompt("format_work_item_description",
		mcp.WithPromptDescription("Format a work item description using proper HTML for Azure DevOps"),
		mcp.WithArgument("description",
			mcp.ArgumentDescription("The description text to format"),
			mcp.RequiredArgument(),
		),
	), handleFormatWorkItemDescription)
}

func handleUpdateWorkItem(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	field := request.Params.Arguments["field"].(string)
	value := request.Params.Arguments["value"].(string)

	// Instead of using a fixed map, directly use the field name
	// This allows any valid Azure DevOps field to be used
	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:    &webapi.OperationValues.Replace,
				Path:  stringPtr("/fields/" + field),
				Value: value,
			},
		},
	}

	workItem, err := workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated work item #%d", *workItem.Id)), nil
}

func handleCreateWorkItem(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workItemType := request.Params.Arguments["type"].(string)
	title := request.Params.Arguments["title"].(string)
	description := request.Params.Arguments["description"].(string)
	priority, hasPriority := request.Params.Arguments["priority"].(string)

	// Create the work item
	createArgs := workitemtracking.CreateWorkItemArgs{
		Type:    &workItemType,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:    &webapi.OperationValues.Add,
				Path:  stringPtr("/fields/System.Title"),
				Value: title,
			},
			{
				Op:    &webapi.OperationValues.Add,
				Path:  stringPtr("/fields/System.Description"),
				Value: description,
			},
		},
	}

	if hasPriority {
		doc := append(*createArgs.Document, webapi.JsonPatchOperation{
			Op:    &webapi.OperationValues.Add,
			Path:  stringPtr("/fields/Microsoft.VSTS.Common.Priority"),
			Value: priority,
		})
		createArgs.Document = &doc
	}

	workItem, err := workItemClient.CreateWorkItem(ctx, createArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create work item: %v", err)), nil
	}

	fields := *workItem.Fields
	var extractedTitle string
	if t, ok := fields["System.Title"].(string); ok {
		extractedTitle = t
	}
	return mcp.NewToolResultText(fmt.Sprintf("Created work item #%d: %s", *workItem.Id, extractedTitle)), nil
}

func handleQueryWorkItems(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.Params.Arguments["query"].(string)

	// Create WIQL query
	wiqlArgs := workitemtracking.QueryByWiqlArgs{
		Wiql: &workitemtracking.Wiql{
			Query: &query,
		},
		// Ensure we pass the project context
		Project: &config.Project,
		// If you have a specific team, you can add it here
		// Team: &teamName,
	}

	queryResult, err := workItemClient.QueryByWiql(ctx, wiqlArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query work items: %v", err)), nil
	}

	// If no work items found, return a message
	if queryResult.WorkItems == nil || len(*queryResult.WorkItems) == 0 {
		return mcp.NewToolResultText("No work items found matching the query."), nil
	}

	// Format results
	var results []string
	
	// If there are many work items, we should limit how many we retrieve details for
	maxDetailsToFetch := 20
	if len(*queryResult.WorkItems) > 0 {
		// Get the first few work item IDs
		count := len(*queryResult.WorkItems)
		if count > maxDetailsToFetch {
			count = maxDetailsToFetch
		}
		
		// Create a list of IDs to fetch
		var ids []int
		for i := 0; i < count; i++ {
			ids = append(ids, *(*queryResult.WorkItems)[i].Id)
		}
		
		// Get the work item details
		if len(ids) > 0 {
			// First add a header line with the total count
			results = append(results, fmt.Sprintf("Found %d work items. Showing details for the first %d:", 
				len(*queryResult.WorkItems), count))
			results = append(results, "")
			
			// Fetch details for these work items
			getArgs := workitemtracking.GetWorkItemsArgs{
				Ids: &ids,
			}
			workItems, err := workItemClient.GetWorkItems(ctx, getArgs)
			if err == nil && workItems != nil && len(*workItems) > 0 {
				for _, item := range *workItems {
					id := *item.Id
					var title, state, workItemType string
					
					if item.Fields != nil {
						if titleVal, ok := (*item.Fields)["System.Title"]; ok {
							title = fmt.Sprintf("%v", titleVal)
						}
						if stateVal, ok := (*item.Fields)["System.State"]; ok {
							state = fmt.Sprintf("%v", stateVal)
						}
						if typeVal, ok := (*item.Fields)["System.WorkItemType"]; ok {
							workItemType = fmt.Sprintf("%v", typeVal)
						}
					}
					
					results = append(results, fmt.Sprintf("ID: %d - [%s] %s (%s)", 
						id, workItemType, title, state))
				}
			} else {
				// Fallback to just listing the IDs if we couldn't get details
				for _, itemRef := range *queryResult.WorkItems {
					results = append(results, fmt.Sprintf("ID: %d", *itemRef.Id))
				}
			}
		}
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

func handleWiqlQueryFormatPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	queryType, exists := request.Params.Arguments["query_type"]
	if !exists {
		return nil, fmt.Errorf("query_type is required")
	}

	additionalFields := request.Params.Arguments["additional_fields"]

	baseFields := "[System.Id], [System.Title], [System.WorkItemType], [System.State], [System.AssignedTo]"
	if additionalFields != "" {
		baseFields += ", " + additionalFields
	}

	var template string
	var explanation string

	switch queryType {
	case "current_sprint":
		template = fmt.Sprintf("SELECT %s FROM WorkItems WHERE [System.IterationPath] = @currentIteration('fanapp')", baseFields)
		explanation = "This query gets all work items in the current sprint. The @currentIteration macro automatically resolves to the current sprint path."

	case "assigned_to_me":
		template = fmt.Sprintf("SELECT %s FROM WorkItems WHERE [System.AssignedTo] = @me AND [System.State] <> 'Closed'", baseFields)
		explanation = "This query gets all active work items assigned to the current user. The @me macro automatically resolves to the current user."

	case "active_bugs":
		template = fmt.Sprintf("SELECT %s FROM WorkItems WHERE [System.WorkItemType] = 'Bug' AND [System.State] <> 'Closed' ORDER BY [Microsoft.VSTS.Common.Priority]", baseFields)
		explanation = "This query gets all active bugs, ordered by priority."

	case "blocked_items":
		template = fmt.Sprintf("SELECT %s FROM WorkItems WHERE [System.State] <> 'Closed' AND [Microsoft.VSTS.Common.Blocked] = 'Yes'", baseFields)
		explanation = "This query gets all work items that are marked as blocked."

	case "recent_activity":
		template = fmt.Sprintf("SELECT %s FROM WorkItems WHERE [System.ChangedDate] > @today-7 ORDER BY [System.ChangedDate] DESC", baseFields)
		explanation = "This query gets all work items modified in the last 7 days, ordered by most recent first."
	}

	return mcp.NewGetPromptResult(
		"WIQL Query Format Helper",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				"system",
				mcp.NewTextContent("You are a WIQL query expert. Help format queries for Azure DevOps work items."),
			),
			mcp.NewPromptMessage(
				"assistant",
				mcp.NewTextContent(fmt.Sprintf("Here's a template for a %s query:\n\n```sql\n%s\n```\n\n%s\n\nCommon WIQL Tips:\n- Use square brackets [] around field names\n- Common macros: @me, @today, @currentIteration\n- Date arithmetic: @today+/-n\n- String comparison is case-insensitive\n- Use 'Contains' for partial matches", queryType, template, explanation)),
			),
		},
	), nil
}

func handleFormatWorkItemDescription(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	description := request.Params.Arguments["description"]
	return mcp.NewGetPromptResult(
		"Azure DevOps Work Item Description Formatter",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				"system",
				mcp.NewTextContent("You format work item descriptions for Azure DevOps. Use proper HTML formatting with <ul>, <li> for bullet points, <p> for paragraphs, and <br> for line breaks."),
			),
			mcp.NewPromptMessage(
				"assistant",
				mcp.NewTextContent(fmt.Sprintf("Here's your description formatted with HTML:\n\n<ul>\n%s\n</ul>",
					strings.Join(strings.Split(description, "-"), "</li>\n<li>"))),
			),
		},
	), nil
}

// Handler for getting detailed work item information
func handleGetWorkItemDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idsStr := request.Params.Arguments["ids"].(string)
	idStrs := strings.Split(idsStr, ",")

	var ids []int
	for _, idStr := range idStrs {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid ID format: %s", idStr)), nil
		}
		ids = append(ids, id)
	}

	workItems, err := workItemClient.GetWorkItems(ctx, workitemtracking.GetWorkItemsArgs{
		Ids:     &ids,
		Project: &config.Project,
		Expand:  &workitemtracking.WorkItemExpandValues.All,
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work items: %v", err)), nil
	}

	var results []string
	for _, item := range *workItems {
		fields := *item.Fields
		title, _ := fields["System.Title"].(string)
		description, _ := fields["System.Description"].(string)
		state, _ := fields["System.State"].(string)

		result := fmt.Sprintf("ID: %d\nTitle: %s\nState: %s\nDescription: %s\n---\n",
			*item.Id, title, state, description)
		results = append(results, result)
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for managing work item relationships
func handleManageWorkItemRelations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceID := int(request.Params.Arguments["source_id"].(float64))
	targetID := int(request.Params.Arguments["target_id"].(float64))
	relationType, ok := request.Params.Arguments["relation_type"].(string)
	if !ok {
		return mcp.NewToolResultError("Invalid relation_type"), nil
	}
	operation := request.Params.Arguments["operation"].(string)

	// Map relation types to Azure DevOps relation types
	relationTypeMap := map[string]string{
		"parent":  "System.LinkTypes.Hierarchy-Reverse",
		"child":   "System.LinkTypes.Hierarchy-Forward",
		"related": "System.LinkTypes.Related",
	}

	azureRelationType := relationTypeMap[relationType]

	var ops []webapi.JsonPatchOperation
	if operation == "add" {
		ops = []webapi.JsonPatchOperation{
			{
				Op:   &webapi.OperationValues.Add,
				Path: stringPtr("/relations/-"),
				Value: map[string]interface{}{
					"rel": azureRelationType,
					"url": fmt.Sprintf("%s/_apis/wit/workItems/%d", config.OrganizationURL, targetID),
					"attributes": map[string]interface{}{
						"comment": "Added via MCP",
					},
				},
			},
		}
	} else {
		// For remove, we need to first get the work item to find the relation index
		workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
			Id:      &sourceID,
			Project: &config.Project,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
		}

		if workItem.Relations == nil {
			return mcp.NewToolResultError("Work item has no relations"), nil
		}

		for i, relation := range *workItem.Relations {
			if *relation.Rel == azureRelationType {
				targetUrl := fmt.Sprintf("%s/_apis/wit/workItems/%d", config.OrganizationURL, targetID)
				if *relation.Url == targetUrl {
					ops = []webapi.JsonPatchOperation{
						{
							Op:   &webapi.OperationValues.Remove,
							Path: stringPtr(fmt.Sprintf("/relations/%d", i)),
						},
					}
					break
				}
			}
		}

		if len(ops) == 0 {
			return mcp.NewToolResultError("Specified relation not found"), nil
		}
	}

	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:       &sourceID,
		Project:  &config.Project,
		Document: &ops,
	}

	_, err := workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update work item relations: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully %sd %s relationship", operation, relationType)), nil
}

// Handler for getting related work items
func handleGetRelatedWorkItems(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	relationType := request.Params.Arguments["relation_type"].(string)

	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Expand:  &workitemtracking.WorkItemExpandValues.Relations,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	if workItem.Relations == nil {
		return mcp.NewToolResultText("No related items found"), nil
	}

	relationTypeMap := map[string]string{
		"parent":   "System.LinkTypes.Hierarchy-Reverse",
		"children": "System.LinkTypes.Hierarchy-Forward",
		"related":  "System.LinkTypes.Related",
	}

	// Debug information
	var debugInfo []string
	debugInfo = append(debugInfo, fmt.Sprintf("Looking for relation type: %s (mapped to: %s)",
		relationType, relationTypeMap[relationType]))

	var relatedIds []int
	for _, relation := range *workItem.Relations {
		debugInfo = append(debugInfo, fmt.Sprintf("Found relation of type: %s", *relation.Rel))

		if relationType == "all" || *relation.Rel == relationTypeMap[relationType] {
			parts := strings.Split(*relation.Url, "/")
			if relatedID, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				relatedIds = append(relatedIds, relatedID)
			}
		}
	}

	if len(relatedIds) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("Debug info:\n%s\n\nNo matching related items found",
			strings.Join(debugInfo, "\n"))), nil
	}

	// Get details of related items
	relatedItems, err := workItemClient.GetWorkItems(ctx, workitemtracking.GetWorkItemsArgs{
		Ids:     &relatedIds,
		Project: &config.Project,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get related items: %v", err)), nil
	}

	var results []string
	for _, item := range *relatedItems {
		fields := *item.Fields
		title, _ := fields["System.Title"].(string)
		result := fmt.Sprintf("ID: %d, Title: %s", *item.Id, title)
		results = append(results, result)
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for adding a comment to a work item
func handleAddWorkItemComment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))
	text := request.Params.Arguments["text"].(string)

	// Add comment as a discussion by updating the Discussion field
	updateArgs := workitemtracking.UpdateWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
		Document: &[]webapi.JsonPatchOperation{
			{
				Op:    &webapi.OperationValues.Add,
				Path:  stringPtr("/fields/System.History"),
				Value: text,
			},
		},
	}

	workItem, err := workItemClient.UpdateWorkItem(ctx, updateArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added comment to work item #%d", *workItem.Id)), nil
}

// Handler for getting work item comments
func handleGetWorkItemComments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["id"].(float64))

	comments, err := workItemClient.GetComments(ctx, workitemtracking.GetCommentsArgs{
		Project:    &config.Project,
		WorkItemId: &id,
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get comments: %v", err)), nil
	}

	var results []string
	for _, comment := range *comments.Comments {
		results = append(results, fmt.Sprintf("Comment by %s at %s:\n%s\n---",
			*comment.CreatedBy.DisplayName,
			comment.CreatedDate.String(),
			*comment.Text))
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for getting work item fields
func handleGetWorkItemFields(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int(request.Params.Arguments["work_item_id"].(float64))

	// Get the work item's details
	workItem, err := workItemClient.GetWorkItem(ctx, workitemtracking.GetWorkItemArgs{
		Id:      &id,
		Project: &config.Project,
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item details: %v", err)), nil
	}

	// Extract and format field information
	var results []string
	fieldName, hasFieldFilter := request.Params.Arguments["field_name"].(string)

	for fieldRef, value := range *workItem.Fields {
		if hasFieldFilter && !strings.Contains(strings.ToLower(fieldRef), strings.ToLower(fieldName)) {
			continue
		}

		results = append(results, fmt.Sprintf("Field: %s\nValue: %v\nType: %T\n---",
			fieldRef,
			value,
			value))
	}

	if len(results) == 0 {
		if hasFieldFilter {
			return mcp.NewToolResultText(fmt.Sprintf("No fields found matching: %s", fieldName)), nil
		}
		return mcp.NewToolResultText("No fields found"), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for batch creating work items
func handleBatchCreateWorkItems(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	itemsJSON := request.Params.Arguments["items"].(string)
	var items []struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority,omitempty"`
	}

	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON format: %v", err)), nil
	}

	var results []string
	for _, item := range items {
		createArgs := workitemtracking.CreateWorkItemArgs{
			Type:    &item.Type,
			Project: &config.Project,
			Document: &[]webapi.JsonPatchOperation{
				{
					Op:    &webapi.OperationValues.Add,
					Path:  stringPtr("/fields/System.Title"),
					Value: item.Title,
				},
				{
					Op:    &webapi.OperationValues.Add,
					Path:  stringPtr("/fields/System.Description"),
					Value: item.Description,
				},
			},
		}

		if item.Priority != "" {
			doc := append(*createArgs.Document, webapi.JsonPatchOperation{
				Op:    &webapi.OperationValues.Add,
				Path:  stringPtr("/fields/Microsoft.VSTS.Common.Priority"),
				Value: item.Priority,
			})
			createArgs.Document = &doc
		}

		workItem, err := workItemClient.CreateWorkItem(ctx, createArgs)
		if err != nil {
			results = append(results, fmt.Sprintf("Failed to create '%s': %v", item.Title, err))
			continue
		}
		results = append(results, fmt.Sprintf("Created work item #%d: %s", *workItem.Id, item.Title))
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Handler for batch updating work items
func handleBatchUpdateWorkItems(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	updatesJSON := request.Params.Arguments["updates"].(string)
	var updates []struct {
		ID    int    `json:"id"`
		Field string `json:"field"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal([]byte(updatesJSON), &updates); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON format: %v", err)), nil
	}

	// Map field names to their System.* equivalents
	fieldMap := map[string]string{
		"Title":       "System.Title",
		"Description": "System.Description",
		"State":       "System.State",
		"Priority":    "Microsoft.VSTS.Common.Priority",
	}

	var results []string
	for _, update := range updates {
		systemField, ok := fieldMap[update.Field]
		if !ok {
			results = append(results, fmt.Sprintf("Invalid field for #%d: %s", update.ID, update.Field))
			continue
		}

		updateArgs := workitemtracking.UpdateWorkItemArgs{
			Id:      &update.ID,
			Project: &config.Project,
			Document: &[]webapi.JsonPatchOperation{
				{
					Op:    &webapi.OperationValues.Replace,
					Path:  stringPtr("/fields/" + systemField),
					Value: update.Value,
				},
			},
		}

		workItem, err := workItemClient.UpdateWorkItem(ctx, updateArgs)
		if err != nil {
			results = append(results, fmt.Sprintf("Failed to update #%d: %v", update.ID, err))
			continue
		}
		results = append(results, fmt.Sprintf("Updated work item #%d", *workItem.Id))
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}
