package droplet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SizesMCPResource represents a handler for MCP Size resources
type SizesMCPResource struct {
	client *godo.Client
}

// NewSizesMCPResource creates a new Sizes MCP resource handler
func NewSizesMCPResource(client *godo.Client) *SizesMCPResource {
	return &SizesMCPResource{
		client: client,
	}
}

// GetResource returns the template for the Sizes MCP resource
func (s *SizesMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"sizes://all",
		"Droplet Sizes",
		mcp.WithResourceDescription("Returns all available droplet sizes"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Sizes MCP resource requests for all sizes
func (s *SizesMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// List all droplet sizes from DigitalOcean API
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200, // Get a large number of sizes at once
	}

	sizes, _, err := s.client.Sizes.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching droplet sizes: %s", err)
	}

	// Serialize to JSON
	// Filter sizes to only include slug, availability, and price details
	filteredSizes := make([]map[string]any, len(sizes))
	for i, size := range sizes {
		filteredSizes[i] = map[string]any{
			"slug":          size.Slug,
			"available":     size.Available,
			"price_monthly": size.PriceMonthly,
			"price_hourly":  size.PriceHourly,
		}
	}

	// Serialize filtered sizes to JSON
	jsonData, err := json.MarshalIndent(filteredSizes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing sizes: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Resources returns the available resources for the Sizes MCP resource
func (s *SizesMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		s.GetResource(): s.HandleGetResource,
	}
}
