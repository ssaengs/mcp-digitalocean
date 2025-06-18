package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// BalanceMCPResource represents a handler for MCP Balance resources
type BalanceMCPResource struct {
	client *godo.Client
}

// NewBalanceMCPResource creates a new Balance MCP resource handler
func NewBalanceMCPResource(client *godo.Client) *BalanceMCPResource {
	return &BalanceMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Balance MCP resource
func (b *BalanceMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"balance://current",
		"Balance Information",
		mcp.WithResourceDescription("Returns balance information"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Balance MCP resource requests
func (b *BalanceMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get balance from DigitalOcean API
	balance, _, err := b.client.Balance.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching balance: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(balance, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing balance: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (b *BalanceMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		b.GetResource(): b.HandleGetResource,
	}
}
