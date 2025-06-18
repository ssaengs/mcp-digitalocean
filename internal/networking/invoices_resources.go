package networking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type InvoicesMCPResource struct {
	client *godo.Client
}

func NewInvoicesMCPResource(client *godo.Client) *InvoicesMCPResource {
	return &InvoicesMCPResource{
		client: client,
	}
}

func (i *InvoicesMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"invoices://list",
		"Invoices",
		mcp.WithResourceDescription("Returns a list of all invoices"),
		mcp.WithMIMEType("application/json"),
	)
}

func (i *InvoicesMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	invoices, _, err := i.client.Invoices.List(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching invoices: %s", err)
	}

	jsonData, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing invoices: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (i *InvoicesMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		i.GetResource(): i.HandleGetResource,
	}
}
