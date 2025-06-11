package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/internal/droplet"
	"regexp"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

type DomainsMCPResource struct {
	client *godo.Client
}

func NewDomainsMCPResource(client *godo.Client) *DomainsMCPResource {
	return &DomainsMCPResource{
		client: client,
	}
}

func (d *DomainsMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"domains://{name}",
		"Domain",
		mcp.WithTemplateDescription("Returns domain information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DomainsMCPResource) GetRecordResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"domains://{name}/records/{record_id}",
		"Domain Record",
		mcp.WithTemplateDescription("Returns information about a domain record"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (d *DomainsMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	domainName, err := extractDomainNameFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid domain URI: %s", err)
	}

	domain, _, err := d.client.Domains.Get(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("error fetching domain: %s", err)
	}

	jsonData, err := json.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing domain: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (d *DomainsMCPResource) HandleGetRecordResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	domainName, recordID, err := extractDomainRecordFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid domain record URI: %s", err)
	}

	record, _, err := d.client.Domains.Record(ctx, domainName, recordID)
	if err != nil {
		return nil, fmt.Errorf("error fetching domain record: %s", err)
	}

	jsonData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing domain record: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func extractDomainNameFromURI(uri string) (string, error) {
	re := regexp.MustCompile(`domains://([^/]+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract domain name from URI: %s", uri)
	}
	return match[1], nil
}

func extractDomainRecordFromURI(uri string) (string, int, error) {
	re := regexp.MustCompile(`domains://([^/]+)/records/(\d+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 3 {
		return "", 0, fmt.Errorf("could not extract domain record from URI: %s", uri)
	}
	recordID, err := strconv.Atoi(match[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid record ID: %s", err)
	}
	return match[1], recordID, nil
}

func (d *DomainsMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]droplet.MCPResourceHandler {
	return map[mcp.ResourceTemplate]droplet.MCPResourceHandler{
		d.GetResourceTemplate():       d.HandleGetResource,
		d.GetRecordResourceTemplate(): d.HandleGetRecordResource,
	}
}
