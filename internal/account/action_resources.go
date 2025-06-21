package account

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-digitalocean/internal/droplet"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// ActionMCPResource represents a handler for MCP Action resources
type ActionMCPResource struct {
	client *godo.Client
}

// NewActionMCPResource creates a new Action MCP resource handler
func NewActionMCPResource(client *godo.Client) *ActionMCPResource {
	return &ActionMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Action MCP resource
func (a *ActionMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"actions://{id}",
		"Action",
		mcp.WithTemplateDescription("Returns action information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the Action MCP resource requests
func (a *ActionMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract action ID from the URI
	actionID, err := extractActionIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid action URI: %s", err)
	}

	// Get action from DigitalOcean API
	action, _, err := a.client.Actions.Get(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("error fetching action: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing action: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (a *ActionMCPResource) Resources() map[mcp.ResourceTemplate]droplet.MCPResourceHandler {
	return map[mcp.ResourceTemplate]droplet.MCPResourceHandler{
		a.GetResourceTemplate(): a.HandleGetResource,
	}
}

// extractActionIDFromURI extracts the action ID from the given URI
func extractActionIDFromURI(uri string) (int, error) {
	var actionID int
	_, err := fmt.Sscanf(uri, "actions://%d", &actionID)
	if err != nil {
		return 0, fmt.Errorf("invalid action URI format: %s", err)
	}
	return actionID, nil
}
