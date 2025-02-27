# MCP Azure DevOps Bridge

A Model Context Protocol (MCP) integration server for Azure DevOps. This focused integration allows you to manage work items, wiki documentation, sprint planning, and handle attachments and discussions seamlessly.

## 🌉 Azure DevOps Integration

Connect with Azure DevOps for comprehensive project management:

- **Work Items** - Create, update, query, and manage work items
- **Wiki Documentation** - Create, update, and retrieve wiki pages
- **Sprint Planning** - Retrieve current sprint information and list sprints
- **Attachments & Discussions** - Add and retrieve attachments and comments to/from work items

## 🚀 Getting Started

### Prerequisites

- Go 1.23 or later
- Azure DevOps Personal Access Token (PAT)

### Installation

#### Installing Go 1.23 or above

##### Windows
Install Go using one of these package managers:

1. Using **winget**:
   ```
   winget install GoLang.Go
   ```

2. Using **Chocolatey**:
   ```
   choco install golang
   ```

3. Using **Scoop**:
   ```
   scoop install go
   ```

After installation, verify with:
```
go version
```

##### macOS
Install Go using Homebrew:

```
brew install go
```

Verify the installation:
```
go version
```

#### Building the Project

1. Clone and build:

```bash
git clone https://github.com/krishh-amilineni/mcp-azuredevops-bridge.git
cd mcp-azuredevops-bridge
go build
```

2. Configure your environment:

```bash
export AZURE_DEVOPS_ORG="your-org"
export AZDO_PAT="your-pat-token"
export AZURE_DEVOPS_PROJECT="your-project"
```

3. Add to Windsurf configuration:

```json
{
  "mcpServers": {
    "azuredevops-bridge": {
      "command": "/full/path/to/mcp-azuredevops-bridge/mcp-azuredevops-bridge",
      "args": [],
      "env": {
        "AZURE_DEVOPS_ORG": "organization",
        "AZDO_PAT": "personal_access_token",
        "AZURE_DEVOPS_PROJECT": "project"
      }
    }
  }
}
```

## 💡 Example Workflows

### Work Item Management

```txt
"Create a user story for the new authentication feature in Azure DevOps"
```

### Wiki Documentation

```txt
"Create a wiki page documenting the API endpoints for our service"
"List all wiki pages in our project wiki"
"Get the content of the 'Getting Started' page from the wiki"
"Show me all available wikis in my Azure DevOps project"
```

### Sprint Planning

```txt
"Show me the current sprint's work items and their status"
```

### Attachments and Comments

```txt
"Add this screenshot as an attachment to work item #123"
```

## 🔧 Features

### Work Item Management
- Create new work items (user stories, bugs, tasks, etc.)
- Update existing work items
- Query work items by various criteria
- Link work items to each other

### Wiki Management
- Create and update wiki pages
- Search wiki content
- Retrieve page content and subpages
- Automatic wiki discovery - dynamically finds all available wikis for your project
- Smart wiki selection - selects the most appropriate wiki based on the project context
- Get list of available wikis for debugging and exploration

### Sprint Management
- Get current sprint information
- List all sprints
- View sprint statistics

### Attachments and Comments
- Add attachments to work items
- Retrieve attachments from work items
- Add comments to work items
- View comments on work items

## 📋 Advanced Wiki Usage

The DevOps Bridge includes enhanced wiki functionality that can help you access documentation more effectively:

### Available Wiki Tools

- `list_wiki_pages` - Lists all wiki pages, optionally from a specific path
- `get_wiki_page` - Retrieves the content of a specific wiki page
- `manage_wiki_page` - Creates or updates a wiki page
- `search_wiki` - Searches for content across wiki pages
- `get_available_wikis` - Lists all available wikis in your Azure DevOps organization

### Wiki Troubleshooting

If you're having trouble accessing wiki content:

1. Use the `get_available_wikis` tool to see all available wikis and their IDs
2. Check that your PAT token has appropriate permissions for wiki access
3. Verify that the wiki path is correct - wiki paths are case-sensitive
4. Enable verbose logging to see detailed request and response information

## 🔒 Security

This integration uses Personal Access Tokens (PAT) for authenticating with Azure DevOps. Ensure your PAT has the appropriate permissions for the operations you want to perform.

## 📝 Credits

This project was inspired by and draws from the original work at [TheApeMachine/mcp-server-devops-bridge](https://github.com/TheApeMachine/mcp-server-devops-bridge). We appreciate their contribution to the open source community.

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
