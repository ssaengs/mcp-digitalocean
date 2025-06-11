package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/droplet"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// VPCMCPResource represents a handler for MCP VPC resources
type VPCMCPResource struct {
	client *godo.Client
}

// NewVPCMCPResource creates a new VPC MCP resource handler
func NewVPCMCPResource(client *godo.Client) *VPCMCPResource {
	return &VPCMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the VPC MCP resource
func (v *VPCMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"vpcs://{id}",
		"VPC",
		mcp.WithTemplateDescription("Returns VPC information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the VPC MCP resource requests
func (v *VPCMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract VPC ID from the URI
	vpcID, err := extractVPCIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid VPC URI: %s", err)
	}

	// Get VPC from DigitalOcean API
	vpc, _, err := v.client.VPCs.Get(ctx, vpcID)
	if err != nil {
		return nil, fmt.Errorf("error fetching VPC: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(vpc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing VPC: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// extractVPCIDFromURI extracts the VPC ID from the URI
func extractVPCIDFromURI(uri string) (string, error) {
	// Use regex to extract the ID from the URI format "vpcs://{id}"
	re := regexp.MustCompile(`vpcs://(.+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract VPC ID from URI: %s", uri)
	}

	return match[1], nil
}

// ResourceTemplates returns the available resource templates for the VPC MCP resource
func (v *VPCMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]droplet.MCPResourceHandler {
	return map[mcp.ResourceTemplate]droplet.MCPResourceHandler{
		v.GetResourceTemplate(): v.HandleGetResource,
	}
}
