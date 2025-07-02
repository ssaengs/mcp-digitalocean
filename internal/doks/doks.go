package doks

import (
	"context"
	"encoding/json"
	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type DoksTool struct {
	client *godo.Client
}

// NewDoksTool creates a new DoksTool instance
func NewDoksTool(client *godo.Client) *DoksTool {
	return &DoksTool{client: client}
}

func (d *DoksTool) GetDoksCluster(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterID := req.GetArguments()["ClusterID"].(string)

	// Get the cluster from DigitalOcean API
	cluster, _, err := d.client.Kubernetes.Get(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// now marshall the cluster spec to JSON
	clusterJSON, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(clusterJSON)), nil
}

func (d *DoksTool) ListDOKSClusters(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := req.GetArguments()["Page"].(int)

	opts := &godo.ListOptions{
		Page: page,
	}

	clusters, _, err := d.client.Kubernetes.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// now marshall the app spec to JSON
	appJSON, err := json.MarshalIndent(clusters, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

// Tools returns the tools provided by the DoksTool
func (d *DoksTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: d.ListDOKSClusters,
			Tool: mcp.NewTool("digitalocean-doks-list-clusters",
				mcp.WithDescription("List all digitalocean Kubernetes clusters"),
				mcp.WithNumber("Page", mcp.Description("The page number to retrieve (default is 1)")),
			),
		},
		{
			Handler: d.GetDoksCluster,
			Tool: mcp.NewTool("digitalocean-doks-get-cluster",
				mcp.WithDescription("Get detailed information about a specific digitalocean Kubernetes cluster"),
				mcp.WithString("ClusterID", mcp.Required(), mcp.Description("The ID of the Kubernetes cluster to retrieve")),
			),
		},
	}
}
