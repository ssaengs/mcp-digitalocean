package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

type PartnerAttachmentMCPResource struct {
	client *godo.Client
}

func NewPartnerAttachmentMCPResource(client *godo.Client) *PartnerAttachmentMCPResource {
	return &PartnerAttachmentMCPResource{
		client: client,
	}
}

func (p *PartnerAttachmentMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"partner_attachment://{id}",
		"Partner Attachment",
		mcp.WithTemplateDescription("Returns partner attachment information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (p *PartnerAttachmentMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	attachmentID, err := extractAttachmentIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid partner attachment URI: %s", err)
	}

	attachment, _, err := p.client.PartnerAttachment.Get(ctx, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("error fetching partner attachment: %s", err)
	}

	jsonData, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing partner attachment: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func extractAttachmentIDFromURI(uri string) (string, error) {
	re := regexp.MustCompile(`partner_attachment://(.+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract partner attachment ID from URI: %s", uri)
	}
	return match[1], nil
}

func (p *PartnerAttachmentMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		p.GetResourceTemplate(): p.HandleGetResource,
	}
}
