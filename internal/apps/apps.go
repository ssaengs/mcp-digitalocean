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
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var create godo.AppCreateRequest
	if err := json.Unmarshal(jsonBytes, &create); err != nil {
		return nil, fmt.Errorf("failed to parse app spec: %w", err)
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

func (a *AppPlatformTool) ListApps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// List all apps using the DigitalOcean API
	page := req.GetArguments()["Page"].(int)
	apps, _, err := a.client.Apps.List(ctx, &godo.ListOptions{Page: page, PerPage: 200})
	if err != nil {
		return nil, err
	}

	// Convert the app information to JSON format
	appsJSON, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(appsJSON)), nil
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

func (a *AppPlatformTool) GetAppUsage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return nil, nil // Not implemented yet
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

type AppUpdate struct {
	Update AppUpdateRequest `json:"update"`
}

// AppUpdateRequest represents the request structure for updating an app
type AppUpdateRequest struct {
	Request *godo.AppUpdateRequest `json:"request"`
	AppID   string                 `json:"app_id"`
}

// UpdateApp updates an existing app by its ID
func (a *AppPlatformTool) UpdateApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(req.GetArguments())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var update AppUpdate
	if err := json.Unmarshal(jsonBytes, &update); err != nil {
		return nil, fmt.Errorf("failed to parse app spec: %w", err)
	}

	// Update the app using the DigitalOcean API
	app, _, err := a.client.Apps.Update(ctx, update.Update.AppID, update.Update.Request)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("App updated successfully: " + app.Spec.Name), nil
}

func (a *AppPlatformTool) Tools() []server.ServerTool {
	tools := []server.ServerTool{
		{
			Handler: a.ListApps,
			Tool: mcp.NewTool("digitalocean-apps-list",
				mcp.WithDescription("List all applications on DigitalOcean App Platform"),
				mcp.WithNumber("Page", mcp.Description("The page number to retrieve (default is 1)")),
			),
		},
		{
			Handler: a.DeleteApp,
			Tool: mcp.NewTool("digitalocean-apps-delete",
				mcp.WithDescription("Delete an existing app"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app we want to delete.")),
			),
		},
		{
			Handler: a.GetAppInfo,
			Tool: mcp.NewTool("digitalocean-apps-get",
				mcp.WithDescription("Get information about an application on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve information for")),
			),
		},
		{
			Handler: a.GetAppUsage,
			Tool: mcp.NewTool("digitalocean-apps-usage",
				mcp.WithDescription("Get usage information for an application on DigitalOcean App Platform"),
				mcp.WithString("AppID", mcp.Required(), mcp.Description("The application ID of the app to retrieve usage information for")),
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

// loadSchema attempts to load the JSON schema from the specified file.
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
