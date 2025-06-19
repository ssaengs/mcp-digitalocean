package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"path/filepath"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AppPlatformTool struct {
	client *godo.Client
}

// NewAppPlatformTool creates a new AppsTool instance
func NewAppPlatformTool(client *godo.Client) (*AppPlatformTool, error) {
	return &AppPlatformTool{client: client}, nil
}

func (a *AppPlatformTool) CreateAppFromAppSpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	if len(jsonBytes) == 0 {
		return nil, fmt.Errorf("len of jsonbytes is 0: %w", jsonBytes)
	}

	// Now we've got the json bytes.
	// we need to serialize them into an AppCreateRequest
	var create godo.AppCreateRequest
	if err := json.Unmarshal(jsonBytes, &create); err != nil {
		return nil, fmt.Errorf("failed to parse app spec: %w", err)
	}

	if create.Spec == nil {
		return nil, fmt.Errorf("app spec is required in the request %+v, %+v", create.Spec, string(jsonBytes))
	}

	// Create the app using the DigitalOcean API
	app, _, err := a.client.Apps.Create(ctx, &create)
	if err != nil {
		return nil, err
	}

	// now marshall the app spec to JSON
	appJSON, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal app spec: %w", err)
	}

	return mcp.NewToolResultText("App created successfully: " + string(appJSON)), nil
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
	appID := req.GetArguments()["AppID"].(string)

	// Get the app using the DigitalOcean API
	app, _, err := a.client.Apps.Get(ctx, appID)
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
	update, ok := req.GetRawArguments().(*godo.AppUpdateRequest)
	if !ok {
		return nil, fmt.Errorf("failed to parse app spec: %v", req.GetRawArguments())
	}

	appID, ok := req.GetArguments()["AppID"].(string)

	// Update the app using the DigitalOcean API
	app, _, err := a.client.Apps.Update(ctx, appID, update)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("App updated successfully: " + app.Spec.Name), nil
}

func (a *AppPlatformTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: a.DeleteApp,
			Tool: mcp.NewTool("digitalocean-apps-delete",
				mcp.WithDescription("Delete an existing app"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID (UUID) of the app we want to delete.")),
			),
		},
		{
			Handler: a.GetAppInfo,
			Tool: mcp.NewTool("digitalocean-apps-get",
				mcp.WithDescription("Get information about an application on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application UUID of the app to retrieve information for")),
			),
		},
	}

	appCreateSchema, err := loadSchema("app-create-schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to generate app create schema: %w", err))
	}

	appCreateTool := server.ServerTool{
		Handler: a.CreateAppFromAppSpec,
		Tool: mcp.NewToolWithRawSchema(
			"digitalocean-create-app-from-spec",
			"Creates an application from a given app spec. Within the app spec, a source has to be provided. The source can be a Git repository, a Dockerfile, or a container image.",
			appCreateSchema,
		),
	}

	appUpdateSchema, err := loadSchema("app-update-schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to generate app create schema: %w", err))
	}

	appUpdateTool := server.ServerTool{
		Handler: a.UpdateApp,
		Tool: mcp.NewToolWithRawSchema(
			"digitalocean-apps-update",
			"Updates an existing application on DigitalOcean App Platform. The app ID and the AppSpec must be provided in the request.",
			appUpdateSchema,
		),
	}

	return append(tools, appCreateTool, appUpdateTool)
}

// loadSchema loads a JSON schema from the specified file in the same directory as the executable.
func loadSchema(file string) ([]byte, error) {
	executablePath, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("failed to get executable path: %w", err))
	}
	executableDir := filepath.Dir(executablePath)

	schema, err := os.ReadFile(filepath.Join(executableDir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", file, err)
	}
	return schema, nil
}
