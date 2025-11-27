//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestCreateApp(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	defer c.Close()

	create := godo.AppCreateRequest{
		Spec: &godo.AppSpec{
			Name: "mcp-e2e-test",
			Services: []*godo.AppServiceSpec{
				{
					Name: "sample-golang",
					GitHub: &godo.GitHubSourceSpec{
						Repo:   "digitalocean/sample-golang",
						Branch: "main",
					},
				},
			},
		},
	}

	resp, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "apps-create-app-from-spec",
			Arguments: create,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError)

	var app godo.App
	appJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(appJSON), &app)
	require.NoError(t, err)
	require.NotEmpty(t, app.ID)

	t.Logf("app ID: %s\n", app.ID)

	// cleanup the app
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "apps-delete",
			Arguments: map[string]interface{}{
				"AppID": app.ID,
			},
		},
	})

	require.NoError(t, err)
	require.False(t, resp.IsError)

	t.Logf("deleted app: %v", app.ID)
}
