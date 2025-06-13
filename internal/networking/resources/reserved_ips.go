package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

type ReservedIPResource struct {
	client *godo.Client
}

func NewReservedIPResource(client *godo.Client) *ReservedIPResource {
	return &ReservedIPResource{
		client: client,
	}
}

func (r *ReservedIPResource) GetIPv4Template() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"reserved_ips://{ip}",
		"Reserved IPv4",
		mcp.WithTemplateDescription("Returns information about a reserved IPv4"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (r *ReservedIPResource) GetIPv6Template() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"reserved_ipv6://{ip}",
		"Reserved IPv6",
		mcp.WithTemplateDescription("Returns information about a reserved IPv6"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (r *ReservedIPResource) HandleGetIPv4(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ip := request.Params.URI[len("reserved_ips://"):]
	reservedIP, _, err := r.client.ReservedIPs.Get(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("error fetching reserved IPv4: %s", err)
	}

	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing reserved IPv4: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (r *ReservedIPResource) HandleGetIPv6(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	ip := request.Params.URI[len("reserved_ipv6://"):]
	reservedIP, _, err := r.client.ReservedIPV6s.Get(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("error fetching reserved IPv6: %s", err)
	}

	jsonData, err := json.MarshalIndent(reservedIP, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing reserved IPv6: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (r *ReservedIPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		r.GetIPv4Template(): r.HandleGetIPv4,
		r.GetIPv6Template(): r.HandleGetIPv6,
	}
}
