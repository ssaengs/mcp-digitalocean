package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// CDNResource represents a handler for MCP CDN resources
type CDNResource struct {
	client *godo.Client
}

// NewCDNResource creates a new CDN MCP resource handler
func NewCDNResource(client *godo.Client) *CDNResource {
	return &CDNResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the CDN MCP resource
func (c *CDNResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"cdn://{id}",
		"CDN",
		mcp.WithTemplateDescription("Returns CDN information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the CDN MCP resource requests
func (c *CDNResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract CDN ID from the URI
	cdnID, err := extractCDNIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid CDN URI: %s", err)
	}

	// Get CDN from DigitalOcean API
	cdn, _, err := c.client.CDNs.Get(ctx, cdnID)
	if err != nil {
		return nil, fmt.Errorf("error fetching CDN: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(cdn, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing CDN: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// extractCDNIDFromURI extracts the CDN ID from the URI
func extractCDNIDFromURI(uri string) (string, error) {
	// Use regex to extract the ID from the URI format "cdn://{id}"
	re := regexp.MustCompile(`cdn://(.+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract CDN ID from URI: %s", uri)
	}

	return match[1], nil
}

// ResourceTemplates returns the available resource templates for the CDN MCP resource
func (c *CDNResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		c.GetResourceTemplate(): c.HandleGetResource,
	}
}
