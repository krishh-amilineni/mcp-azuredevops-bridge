package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/wiki"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// AzureDevOpsConfig holds the configuration for Azure DevOps connection
type AzureDevOpsConfig struct {
	OrganizationURL     string
	PersonalAccessToken string
	Project             string
}

// Global clients and config
var (
	connection     *azuredevops.Connection
	workItemClient workitemtracking.Client
	wikiClient     wiki.Client
	coreClient     core.Client
	config         AzureDevOpsConfig
)

func main() {
	// Main function for the MCP server - handles initialization and startup
	// Load configuration from environment variables
	config = AzureDevOpsConfig{
		OrganizationURL:     "https://dev.azure.com/" + os.Getenv("AZURE_DEVOPS_ORG"),
		PersonalAccessToken: os.Getenv("AZDO_PAT"),
		Project:             os.Getenv("AZURE_DEVOPS_PROJECT"),
	}

	// Validate configuration
	if config.OrganizationURL == "" || config.PersonalAccessToken == "" || config.Project == "" {
		log.Fatal("Missing required environment variables: AZURE_DEVOPS_ORG, AZDO_PAT, AZURE_DEVOPS_PROJECT")
	}

	// Initialize Azure DevOps clients
	if err := initializeClients(config); err != nil {
		log.Fatalf("Failed to initialize Azure DevOps clients: %v", err)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"MCP Azure DevOps Bridge",
		"1.0.0",
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Configure custom error handling
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(&logWriter{})

	// Add Work Item tools
	addWorkItemTools(s)

	// Add Wiki tools
	addWikiTools(s)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func stringPtr(s string) *string {
	return &s
}

// Initialize Azure DevOps clients
func initializeClients(config AzureDevOpsConfig) error {
	connection = azuredevops.NewPatConnection(config.OrganizationURL, config.PersonalAccessToken)

	ctx := context.Background()

	var err error

	// Initialize Work Item Tracking client
	workItemClient, err = workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to create work item client: %v", err)
	}

	// Initialize Wiki client
	wikiClient, err = wiki.NewClient(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to create wiki client: %v", err)
	}

	// Initialize Core client
	coreClient, err = core.NewClient(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to create core client: %v", err)
	}

	return nil
}

type logWriter struct{}

func (w *logWriter) Write(bytes []byte) (int, error) {
	// Skip logging "Prompts not supported" errors
	if strings.Contains(string(bytes), "Prompts not supported") {
		return len(bytes), nil
	}
	return fmt.Print(string(bytes))
}
