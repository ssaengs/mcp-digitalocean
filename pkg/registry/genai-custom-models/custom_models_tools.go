package genaicustommodels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const customModelsAPIPath = "v2/gen-ai/custom_models"

// CustomModelsTool provides custom model management tools.
type CustomModelsTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewCustomModelsTool creates a new CustomModelsTool instance.
func NewCustomModelsTool(client func(ctx context.Context) (*godo.Client, error)) *CustomModelsTool {
	return &CustomModelsTool{client: client}
}

// newRequestWithContext builds an authenticated HTTP request via the godo client.
func newRequestWithContext(ctx context.Context, client *godo.Client, method, urlPath string, body interface{}) (*http.Request, error) {
	req0, err := client.NewRequest(ctx, method, urlPath, body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, req0.Method, req0.URL.String(), req0.Body)
	if err != nil {
		return nil, err
	}
	req.Header = req0.Header.Clone()
	if req0.ContentLength > 0 {
		req.ContentLength = req0.ContentLength
	}
	if req0.GetBody != nil {
		req.GetBody = req0.GetBody
	}
	return req, nil
}

// listModels lists custom models with optional filters.
func (cmt *CustomModelsTool) listModels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	args := req.GetArguments()
	q := url.Values{}
	if status, ok := args["status"].(string); ok && status != "" {
		q.Set("status", status)
	}
	if page, ok := args["page"].(float64); ok {
		q.Set("page", fmt.Sprintf("%d", int(page)))
	}
	if perPage, ok := args["per_page"].(float64); ok {
		q.Set("per_page", fmt.Sprintf("%d", int(perPage)))
	}

	path := customModelsAPIPath
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	apiReq, err := newRequestWithContext(ctx, client, "GET", path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListCustomModelsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list custom models", err), nil
	}

	type ListResponse struct {
		Models       []*CustomModel `json:"models"`
		Count        int            `json:"count"`
		MaxThreshold int            `json:"max_threshold,omitempty"`
	}

	response := ListResponse{
		Models:       output.Models,
		Count:        len(output.Models),
		MaxThreshold: output.MaxThreshold,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// importModel imports a custom model from an external source.
func (cmt *CustomModelsTool) importModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	sourceType, _ := args["source_type"].(string)
	if sourceType == "" {
		return mcp.NewToolResultError("source_type is required"), nil
	}

	sourceRefRaw, ok := args["source_ref"].(map[string]interface{})
	if !ok || sourceRefRaw == nil {
		return mcp.NewToolResultError("source_ref is required"), nil
	}

	sourceRef := CustomModelSourceRef{}
	if v, ok := sourceRefRaw["repo_id"].(string); ok {
		sourceRef.RepoID = v
	}
	if v, ok := sourceRefRaw["commit_sha"].(string); ok {
		sourceRef.CommitSHA = v
	}
	if v, ok := sourceRefRaw["access_type"].(string); ok {
		sourceRef.AccessType = CustomModelAccessType(v)
	}
	if v, ok := sourceRefRaw["hf_token"].(string); ok {
		sourceRef.HFToken = v
	}

	acceptTerms, _ := args["accept_terms_and_conditions"].(bool)

	input := &ImportCustomModelInput{
		Name:                     name,
		SourceType:               CustomModelSourceType(sourceType),
		SourceRef:                sourceRef,
		AcceptTermsAndConditions: acceptTerms,
	}

	if desc, ok := args["description"].(string); ok && desc != "" {
		input.Description = desc
	}
	if region, ok := args["preferred_gpu_region"].(string); ok && region != "" {
		input.PreferredGPURegion = region
	}
	if tagsRaw, ok := args["tags"].(map[string]interface{}); ok {
		if tagsList, ok := tagsRaw["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsList {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
			input.Tags = &CustomModelTags{Tags: tags}
		}
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newRequestWithContext(ctx, client, "POST", customModelsAPIPath+"/import", input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ImportCustomModelOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to import custom model", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateMetadata updates the metadata of a custom model.
func (cmt *CustomModelsTool) updateMetadata(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	uuid, _ := args["uuid"].(string)
	if uuid == "" {
		return mcp.NewToolResultError("uuid is required"), nil
	}

	input := &UpdateCustomModelMetadataInput{}
	hasUpdate := false

	if name, ok := args["name"].(string); ok && name != "" {
		input.Name = &name
		hasUpdate = true
	}
	if desc, ok := args["description"].(string); ok && desc != "" {
		input.Description = &desc
		hasUpdate = true
	}
	if tagsRaw, ok := args["tags"].(map[string]interface{}); ok {
		if tagsList, ok := tagsRaw["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsList {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
			input.Tags = &CustomModelTags{Tags: tags}
			hasUpdate = true
		}
	}

	if !hasUpdate {
		return mcp.NewToolResultError("at least one of name, description, or tags must be provided"), nil
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newRequestWithContext(ctx, client, "PATCH", customModelsAPIPath+"/"+uuid+"/metadata", input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output UpdateCustomModelMetadataOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to update custom model metadata", err), nil
	}

	jsonData, err := json.MarshalIndent(output.Model, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// deleteModel deletes a custom model.
func (cmt *CustomModelsTool) deleteModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uuid, _ := req.GetArguments()["uuid"].(string)
	if uuid == "" {
		return mcp.NewToolResultError("uuid is required"), nil
	}

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newRequestWithContext(ctx, client, "DELETE", customModelsAPIPath+"/"+uuid, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output DeleteCustomModelOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to delete custom model", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for custom model management.
func (cmt *CustomModelsTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: cmt.listModels,
			Tool: mcp.NewTool(
				"genai-custom-models-list",
				mcp.WithDescription("List custom models with optional status filter and pagination."),
				mcp.WithString("status", mcp.Description("Filter by status: STATUS_IMPORTING, STATUS_READY, STATUS_FAILED, STATUS_DELETED")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page (default: 20)")),
			),
		},
		{
			Handler: cmt.importModel,
			Tool: mcp.NewTool(
				"genai-custom-models-import",
				mcp.WithDescription("Import a custom model from an external source (e.g. HuggingFace). Starts an async import job."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the custom model")),
				mcp.WithString("source_type", mcp.Required(), mcp.Description("Source type: SOURCE_TYPE_HUGGINGFACE, SOURCE_TYPE_SPACES_BUCKET, SOURCE_TYPE_SDK_UPLOAD, SOURCE_TYPE_FINE_TUNING")),
				mcp.WithObject("source_ref", mcp.Required(), mcp.Description("Source reference: repo_id (string), commit_sha (string, optional), access_type (ACCESS_TYPE_PUBLIC, ACCESS_TYPE_PRIVATE, ACCESS_TYPE_GATED), hf_token (string, for private/gated models)")),
				mcp.WithBoolean("accept_terms_and_conditions", mcp.Description("Accept terms and conditions for importing the model")),
				mcp.WithString("description", mcp.Description("Description of the model")),
				mcp.WithString("preferred_gpu_region", mcp.Description("Preferred GPU region for the model (e.g. nyc3)")),
				mcp.WithObject("tags", mcp.Description("Tags object with a 'tags' array of strings")),
			),
		},
		{
			Handler: cmt.updateMetadata,
			Tool: mcp.NewTool(
				"genai-custom-models-update-metadata",
				mcp.WithDescription("Update the name, description, or tags of an existing custom model."),
				mcp.WithString("uuid", mcp.Required(), mcp.Description("UUID of the custom model to update")),
				mcp.WithString("name", mcp.Description("New name for the model")),
				mcp.WithString("description", mcp.Description("New description for the model")),
				mcp.WithObject("tags", mcp.Description("New tags object with a 'tags' array of strings")),
			),
		},
		{
			Handler: cmt.deleteModel,
			Tool: mcp.NewTool(
				"genai-custom-models-delete",
				mcp.WithDescription("Delete a custom model."),
				mcp.WithString("uuid", mcp.Required(), mcp.Description("UUID of the custom model to delete")),
			),
		},
	}
}
