package apps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AppPlatformTool struct {
	client *godo.Client

	// appCreateSchema and appUpdateSchema are JSON schemas for creating and updating apps. Wrap the spec and the optional arguments in a JSON schema to work around the limitations of not
	// being able to use ToolOptions when creating tools with mcp.NewToolWithRawSchema()
	appCreateSchema []byte
	appUpdateSchema []byte
}

// NewAppPlatformTool creates a new AppsTool instance
func NewAppPlatformTool(client *godo.Client) (*AppPlatformTool, error) {
	reflector := jsonschema.Reflector{}
	err := reflector.AddGoComments("github.com/digitalocean/godo", "./apps.gen.go")
	if err != nil {
		return nil, fmt.Errorf("failed to add go comments: %w", err)
	}

	appCreateSchema, err := reflector.Reflect(&appCreateRequest{}).MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal appCreateRequest schema: %w", err)
	}

	appUpdateSchema, err := reflector.Reflect(&appUpdateRequest{}).MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal appCreateRequest schema: %w", err)
	}

	return &AppPlatformTool{
		client:          client,
		appCreateSchema: appCreateSchema,
		appUpdateSchema: appUpdateSchema,
	}, nil
}

type appCreateRequest struct {
	Spec *godo.AppSpec `json:"spec"`
	// ProjectID is optional and can be used to specify the project under which the app should be created
	ProjectID string `json:"project_id,omitempty"`
}

type appUpdateRequest struct {
	Spec *godo.AppSpec `json:"spec"`
	// AppID is the ID of the app to update
	AppID string `json:"app_id,omitempty"`
}

func (a *AppPlatformTool) CreateAppFromGit(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	create, ok := req.GetRawArguments().(*appUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("failed to parse app spec")
	}

	// Create the app using the DigitalOcean API
	app, _, err := a.client.Apps.Create(ctx, &godo.AppCreateRequest{
		Spec: create.Spec,
	})
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("App created successfully: " + app.Spec.Name), nil
}

// DeleteApp deletes an existing app by its ID
func (a *AppPlatformTool) DeleteApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the app ID from the request
	appID := req.GetArguments()["AppID"].(string)

	// Delete the app using the DigitalOcean API
	_, err := a.client.Apps.Delete(ctx, appID)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("App deleted successfully"), nil
}

// GetAppInfo retrieves an app by its ID
func (a *AppPlatformTool) GetAppInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the app ID from the request

	// Get the app using the DigitalOcean API
	app, _, err := a.client.Apps.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	// Convert the app information to JSON format
	appJSON, err := json.MarshalIndent(app.Spec, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(appJSON)), nil
}

// UpdateApp updates an existing app by its ID
func (a *AppPlatformTool) UpdateApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	update, ok := req.GetRawArguments().(*appUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("failed to parse app spec")
	}

	appID, ok := req.GetArguments()["AppID"].(string)

	// Create the update request
	updateRequest := &godo.AppUpdateRequest{
		Spec: update.Spec,
	}

	// Update the app using the DigitalOcean API
	app, _, err := a.client.Apps.Update(ctx, appID, updateRequest)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("App updated successfully: " + app.Spec.Name), nil
}

// Tools returns the tools provided by the AppsTool
func (a *AppPlatformTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: a.CreateAppFromGit,
			Tool: mcp.NewToolWithRawSchema(
				"create-app",
				"Create a new app by submitting an app specification. For documentation\n  on app specifications (\"AppSpec\" objects),"+
					" please refer to [the product\n    documentation](https://docs.digitalocean.com/products/app-platform/reference/app-spec/).",
				a.appCreateSchema,
			),
		},
		{
			Handler: a.DeleteApp,
			Tool: mcp.NewTool("apps-delete",
				mcp.WithDescription("Delete an existing app"),
				// Define the parameters required for this tool
				mcp.WithString("AppID", mcp.Required(), mcp.Description("ID of the app to delete")),
			),
		},
		{
			Handler: a.GetAppInfo,
			Tool: mcp.NewTool("apps-get-info",
				mcp.WithDescription("Get information about an app"),
				// Define the parameters required for this tool
				mcp.WithString("AppID", mcp.Required(), mcp.Description("ID of the app to retrieve information for")),
			),
		},
		{
			Handler: a.UpdateApp,
			Tool: mcp.NewToolWithRawSchema(
				"apps-update",
				"Update an existing app by submitting a new app specification. "+
					"For\n    documentation on app specifications (\"AppSpec\" objects),"+
					" please refer to\n    [the product documentation](https://docs.digitalocean.com/products/app-platform/reference/app-spec/).",
				a.appUpdateSchema,
			),
		},
	}
}
