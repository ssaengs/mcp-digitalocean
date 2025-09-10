package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type UserTool struct {
	client func(ctx context.Context) *godo.Client
}

func NewUserTool(client func(ctx context.Context) *godo.Client) *UserTool {
	return &UserTool{
		client: client,
	}
}

func (s *UserTool) getUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	user, ok := args["user"].(string)
	if !ok || user == "" {
		return mcp.NewToolResultError("User name is required"), nil
	}

	dbUser, _, err := s.client(ctx).Databases.GetUser(ctx, id, user)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUser, err := json.MarshalIndent(dbUser, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return mcp.NewToolResultText(string(jsonUser)), nil
}

func (s *UserTool) listUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}

	page := 0
	if pStr, ok := args["page"].(string); ok && pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			page = p
		}
	}

	perPage := 0
	if pp, ok := args["per_page"].(int); ok {
		perPage = pp
	}

	var opts *godo.ListOptions
	if page > 0 || perPage > 0 {
		opts = &godo.ListOptions{Page: page, PerPage: perPage}
	}

	users, _, err := s.client(ctx).Databases.ListUsers(ctx, id, opts)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUsers, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUsers)), nil
}

func (s *UserTool) createUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("User name is required"), nil
	}

	createReq := &godo.DatabaseCreateUserRequest{Name: name}

	if plugin, ok := args["mysql_auth_plugin"].(string); ok && plugin != "" {
		createReq.MySQLSettings = &godo.DatabaseMySQLUserSettings{AuthPlugin: plugin}
	}

	if settingsVal, ok := args["settings"]; ok {
		settingsMap, ok := settingsVal.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("Invalid settings object: must be an object"), nil
		}
		settingsBytes, _ := json.Marshal(settingsMap)
		var settings godo.DatabaseUserSettings
		if err := json.Unmarshal(settingsBytes, &settings); err != nil {
			return mcp.NewToolResultError("Invalid settings object: " + err.Error()), nil
		}
		createReq.Settings = &settings
	}

	// Nil check for s.client(ctx).Databases after argument validation
	if s.client == nil || s.client(ctx).Databases == nil {
		return mcp.NewToolResultError("internal error: database client is not configured"), nil
	}

	dbUser, _, err := s.client(ctx).Databases.CreateUser(ctx, id, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUser, err := json.MarshalIndent(dbUser, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUser)), nil
}

func (s *UserTool) updateUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	user, ok := args["user"].(string)
	if !ok || user == "" {
		return mcp.NewToolResultError("User name is required"), nil
	}

	updateReq := &godo.DatabaseUpdateUserRequest{}

	if settingsVal, ok := args["settings"]; ok {
		settingsMap, ok := settingsVal.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("Invalid settings object: must be an object"), nil
		}
		settingsBytes, _ := json.Marshal(settingsMap)
		var settings godo.DatabaseUserSettings
		if err := json.Unmarshal(settingsBytes, &settings); err != nil {
			return mcp.NewToolResultError("Invalid settings object: " + err.Error()), nil
		}
		updateReq.Settings = &settings
	}

	// Nil check for s.client(ctx).Databases after argument validation and settings validation
	if s.client == nil || s.client(ctx).Databases == nil {
		return mcp.NewToolResultError("internal error: database client is not configured"), nil
	}

	dbUser, _, err := s.client(ctx).Databases.UpdateUser(ctx, id, user, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonUser, err := json.MarshalIndent(dbUser, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonUser)), nil
}

func (s *UserTool) deleteUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("Cluster id is required"), nil
	}
	user, ok := args["user"].(string)
	if !ok || user == "" {
		return mcp.NewToolResultError("User name is required"), nil
	}

	_, err := s.client(ctx).Databases.DeleteUser(ctx, id, user)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	return mcp.NewToolResultText("User deleted successfully"), nil
}

func (s *UserTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: s.getUser,
			Tool: mcp.NewTool("db-cluster-get-user",
				mcp.WithDescription("Get a database user by cluster id and user name"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster ID")),
				mcp.WithString("user", mcp.Required(), mcp.Description("The user name")),
			),
		},
		{
			Handler: s.listUsers,
			Tool: mcp.NewTool("db-cluster-list-users",
				mcp.WithDescription("List database users for a cluster"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster ID")),
				mcp.WithString("page", mcp.Description("Page number for pagination (optional)")),
				mcp.WithNumber("per_page", mcp.Description("Number of results per page (optional)")),
			),
		},
		{
			Handler: s.createUser,
			Tool: mcp.NewTool("db-cluster-create-user",
				mcp.WithDescription("Create a new database user for a cluster"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster ID")),
				mcp.WithString("name", mcp.Required(), mcp.Description("The user name")),
				mcp.WithString("mysql_auth_plugin", mcp.Description("MySQL auth plugin (optional)")),
				mcp.WithObject("settings",
					mcp.Description("Optional user settings object"),
					mcp.Properties(map[string]any{
						"acl": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id":         map[string]any{"type": "string"},
									"permission": map[string]any{"type": "string"},
									"topic":      map[string]any{"type": "string"},
								},
							},
						},
						"opensearch_acl": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"index":      map[string]any{"type": "string"},
									"permission": map[string]any{"type": "string"},
								},
							},
						},
						"mongo_user_settings": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"databases": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "string"},
								},
								"role": map[string]any{"type": "string"},
							},
						},
					}),
				),
			),
		},
		{
			Handler: s.updateUser,
			Tool: mcp.NewTool("db-cluster-update-user",
				mcp.WithDescription("Update a database user's settings"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster ID")),
				mcp.WithString("user", mcp.Required(), mcp.Description("The user name")),
				mcp.WithObject("settings",
					mcp.Description("Optional user settings object"),
					mcp.Properties(map[string]any{
						"acl": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id":         map[string]any{"type": "string"},
									"permission": map[string]any{"type": "string"},
									"topic":      map[string]any{"type": "string"},
								},
							},
						},
						"opensearch_acl": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"index":      map[string]any{"type": "string"},
									"permission": map[string]any{"type": "string"},
								},
							},
						},
						"mongo_user_settings": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"databases": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "string"},
								},
								"role": map[string]any{"type": "string"},
							},
						},
					}),
				),
			),
		},
		{
			Handler: s.deleteUser,
			Tool: mcp.NewTool("db-cluster-delete-user",
				mcp.WithDescription("Delete a database user from a cluster"),
				mcp.WithString("id", mcp.Required(), mcp.Description("The cluster ID")),
				mcp.WithString("user", mcp.Required(), mcp.Description("The user name to delete")),
			),
		},
	}
}
