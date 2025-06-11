package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/droplet"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// FirewallMCPResource represents a handler for MCP Firewall resources
type FirewallMCPResource struct {
	client *godo.Client
}

// NewFirewallMCPResource creates a new Firewall MCP resource handler
func NewFirewallMCPResource(client *godo.Client) *FirewallMCPResource {
	return &FirewallMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Firewall MCP resource
func (f *FirewallMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"firewalls://{id}",
		"Firewall",
		mcp.WithTemplateDescription("Returns firewall information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the Firewall MCP resource requests
func (f *FirewallMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	firewallID := request.Params.URI[len("firewalls://"):]
	firewall, _, err := f.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		return nil, fmt.Errorf("error fetching firewall: %s", err)
	}

	jsonData, err := json.MarshalIndent(firewall, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing firewall: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// ResourceTemplates returns the available resource templates for the Firewall MCP resource
func (f *FirewallMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]droplet.MCPResourceHandler {
	return map[mcp.ResourceTemplate]droplet.MCPResourceHandler{
		f.GetResourceTemplate(): f.HandleGetResource,
	}
}
