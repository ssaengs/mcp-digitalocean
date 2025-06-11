package droplet

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ImagesMCPResource represents a handler for MCP Images response
type ImagesMCPResource struct {
	client *godo.Client
}

// NewImagesMCPResource creates a new Images MCP resource handler
func NewImagesMCPResource(client *godo.Client) *ImagesMCPResource {
	return &ImagesMCPResource{
		client: client,
	}
}

// We will provide a general resource which will get all images
// Another template resource that will provide a detailed single image.

// GetResource returns the template for the Images MCP resource
func (i *ImagesMCPResource) GetResource() mcp.Resource {
	return mcp.NewResource(
		"images://distribution",
		"Distribution Images",
		mcp.WithResourceDescription("Returns all available distribution images"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleGetResource handles the Images MCP resource requests for all images
func (i *ImagesMCPResource) HandleGetResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// List all images from DigitalOcean API
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200, // Get a large number of images at once
	}

	images, _, err := i.client.Images.ListDistribution(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error fetching images: %s", err)
	}

	// Serialize to JSON
	// Filter images to only include id, name, distribution, and type details
	filteredImages := make([]map[string]any, len(images))
	for i, image := range images {
		filteredImages[i] = map[string]any{
			"id":           image.ID,
			"name":         image.Name,
			"distribution": image.Distribution,
			"type":         image.Type,
		}
	}

	// Serialize filtered images to JSON
	jsonData, err := json.MarshalIndent(filteredImages, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing images: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// GetResourceTemplate returns the template for the Images MCP resource
func (i *ImagesMCPResource) GetResourceTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"images://{id}",
		"Image",
		mcp.WithTemplateDescription("Returns image information"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandleGetResourceTemplates handles the Images MCP resource requests
func (i *ImagesMCPResource) HandleGetResourceTemplates(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract image ID from the URI
	imageID, err := extractImageIDFromURI(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid image URI: %s", err)
	}

	// Get image from DigitalOcean API
	image, _, err := i.client.Images.GetByID(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching image: %s", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(image, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing image: %s", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// extractImageIDFromURI extracts the image ID from the URI
func extractImageIDFromURI(uri string) (int, error) {
	// Extract the image ID from the URI
	// The URI format is assumed to be "images://{id}"
	// Split the URI by "//" to get the image ID
	parts := strings.Split(uri, "//")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid image URI format")
	}

	imageID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid image ID: %s", err)
	}

	return imageID, nil
}

// Resources returns the resources for the Images MCP resource
func (i *ImagesMCPResource) Resources() map[mcp.Resource]server.ResourceHandlerFunc {
	return map[mcp.Resource]server.ResourceHandlerFunc{
		i.GetResource(): i.HandleGetResource,
	}
}

// ResourceTemplates returns the available resources for the Images MCP resource
func (i *ImagesMCPResource) ResourceTemplates() map[mcp.ResourceTemplate]MCPResourceHandler {
	return map[mcp.ResourceTemplate]MCPResourceHandler{
		i.GetResourceTemplate(): i.HandleGetResource,
	}
}
