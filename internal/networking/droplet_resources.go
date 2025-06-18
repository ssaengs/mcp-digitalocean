package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

type MCPResourceHandler = func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error)

// DropletMCPResource represents a handler for MCP Droplet resources
type DropletMCPResource struct {
	client *godo.Client
}

// NewDropletMCPResource creates a new Droplet MCP resource handler
func NewDropletMCPResource(client *godo.Client) *DropletMCPResource {
	return &DropletMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Droplet MCP resource
func (d *DropletMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"droplets://{id}",
		"Droplet",
		mcp.WithTemplateDescription("Returns droplet information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the Droplet MCP resource requests
func (d *DropletMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract droplet ID from the URI
	dropletID, err := extractDropletIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid droplet URI: %s", err)
	}

	// Get droplet from DigitalOcean API
	droplet, _, err := d.client.Droplets.Get(ctx, dropletID)
	if err != nil {
		return nil, fmt.Errorf("error fetching droplet: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(droplet, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing droplet: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// extractDropletIDFromURI extracts the droplet ID from the URI
func extractDropletIDFromURI(uri string) (int, error) {
	// Use regex to extract the ID from the URI format "droplets://{id}"
	re := regexp.MustCompile(`droplets://(\d+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return 0, fmt.Errorf("could not extract droplet ID from URI: %s", uri)
	}

	// Convert the ID to an integer
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("invalid droplet ID: %s", err)
	}

	return id, nil
}

// GetActionsResource returns a template for droplet actions
func (d *DropletMCPResource) GetActionsResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"droplets://{id}/actions/{action_id}",
		"Droplet Action",
		mcp.WithTemplateDescription("Returns information about a droplet action"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleActionsResource handles requests for droplet actions
func (d *DropletMCPResource) HandleGetActionsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract IDs from the URI
	uri := request.Params.URI
	parts := strings.Split(uri, "/")

	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid action URI format: %s", uri)
	}

	// Extract droplet ID
	dropletIDStr := strings.TrimPrefix(parts[2], "droplets://")
	dropletID, err := strconv.Atoi(dropletIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid droplet ID: %s", err)
	}

	// Extract action ID
	actionID, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid action ID: %s", err)
	}

	// Get action from DigitalOcean API
	action, _, err := d.client.DropletActions.Get(ctx, dropletID, actionID)
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
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// ResourceTemplates returns the available resource templates for the Droplet MCP resource
func (d *DropletMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		d.GetResourceTemplate():        d.HandleGetResource,
		d.GetActionsResourceTemplate(): d.HandleGetActionsResource,
	}
}
