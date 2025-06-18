package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegionsMCPResource represents a handler for MCP Regions response
type RegionsMCPResource struct {
	client *godo.Client
}

// NewRegionsMCPResource creates a new Regions MCP resource handler
func NewRegionsMCPResource(client *godo.Client) *RegionsMCPResource {
	return &RegionsMCPResource{
		client: client,
	}
}

// GetResource returns the template for the Regions MCP resource
func (r *RegionsMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"regions://all",
		"Regions",
		mcp.WithResourceDescription("Returns all available regions"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Regions MCP resource requests for all regions
func (r *RegionsMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// List all regions from DigitalOcean API
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	regions, _, err := r.client.Regions.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(regions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing regions: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Resources returns the resources for the Regions MCP resource
func (r *RegionsMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		r.GetResource(): r.HandleGetResource,
	}
}
