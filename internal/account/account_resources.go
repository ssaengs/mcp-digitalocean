package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AccountMCPResource struct {
	client *godo.Client
}

func NewAccountMCPResource(client *godo.Client) *AccountMCPResource {
	return &AccountMCPResource{
		client: client,
	}
}

func (a *AccountMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"account://current",
		"Account Information",
		mcp.WithResourceDescription("Returns account information"),
		mcp.WithMIMEType("application/json"),
	)
}

func (a *AccountMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get account information from DigitalOcean API
	account, _, err := a.client.Account.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching account: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing account: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (a *AccountMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		a.GetResource(): a.HandleGetResource,
	}
}
