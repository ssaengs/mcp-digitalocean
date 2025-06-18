package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

// CertificateMCPResource represents a handler for MCP Certificate resources
type CertificateMCPResource struct {
	client *godo.Client
}

// NewCertificateMCPResource creates a new Certificate MCP resource handler
func NewCertificateMCPResource(client *godo.Client) *CertificateMCPResource {
	return &CertificateMCPResource{
		client: client,
	}
}

// GetResourceTemplate returns the template for the Certificate MCP resource
func (c *CertificateMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"certificates://{id}",
		"Certificate",
		mcp.WithTemplateDescription("Returns certificate information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResource handles the Certificate MCP resource requests
func (c *CertificateMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	certID := request.Params.URI[len("certificates://"):]
	certificate, _, err := c.client.Certificates.Get(ctx, certID)
	if err != nil {
		return nil, fmt.Errorf("error fetching certificate: %s", err)
	}

	jsonData, err := json.MarshalIndent(certificate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing certificate: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// ResourceTemplates returns the available resource templates for the Certificate MCP resource
func (c *CertificateMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		c.GetResourceTemplate(): c.HandleGetResource,
	}
}
