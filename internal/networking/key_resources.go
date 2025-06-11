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

type KeysMCPResource struct {
	client *godo.Client
}

func NewKeysMCPResource(client *godo.Client) *KeysMCPResource {
	return &KeysMCPResource{
		client: client,
	}
}

func (k *KeysMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"keys://{id}",
		"SSH Key",
		mcp.WithTemplateDescription("Returns SSH key information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (k *KeysMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	keyID, err := extractKeyIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid key URI: %s", err)
	}

	key, _, err := k.client.Keys.GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("error fetching key: %s", err)
	}

	jsonKey, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing key: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonKey),
		},
	}, nil
}

func extractKeyIDFromURI(uri string) (int, error) {
	re := regexp.MustCompile(`keys://(\d+)`)
	match := re.FindStringSubmatch(uri)
	if len(match) < 2 {
		return 0, fmt.Errorf("could not extract key ID from URI: %s", uri)
	}

	id, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("invalid key ID: %s", err)
	}

	return id, nil
}

func (k *KeysMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]droplet.MCPResourceHandler {
	return map[mcp.ResourceTemplate]droplet.MCPResourceHandler{
		k.GetResourceTemplate(): k.HandleGetResource,
	}
}
