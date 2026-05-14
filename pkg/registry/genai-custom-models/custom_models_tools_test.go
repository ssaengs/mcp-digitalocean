package genaicustommodels

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func setupCustomModelsToolWithFailingClient() *CustomModelsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return nil, context.Canceled
	}
	return NewCustomModelsTool(client)
}

func TestCustomModelsTool_Tools(t *testing.T) {
	tool := NewCustomModelsTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{}, nil
	})

	tools := tool.Tools()
	require.Len(t, tools, 6, "should have 6 custom models tools")

	expectedTools := map[string]bool{
		"genai-custom-models-list":            false,
		"genai-custom-models-import":          false,
		"genai-custom-models-update-metadata": false,
		"genai-custom-models-get":             false,
		"genai-custom-models-delete":          false,
		"genai-models-unified-search":         false,
	}

	for _, st := range tools {
		name := st.Tool.Name
		_, ok := expectedTools[name]
		require.True(t, ok, "unexpected tool name: %s", name)
		expectedTools[name] = true
	}

	for name, found := range expectedTools {
		require.True(t, found, "missing expected tool: %s", name)
	}
}

func TestCustomModelsTool_listModels_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.listModels(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_importModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing name", args: map[string]any{
			"source_type": "SOURCE_TYPE_HUGGINGFACE",
			"source_ref":  map[string]interface{}{"repo_id": "test/model"},
		}},
		{name: "missing source_type", args: map[string]any{
			"name":       "my-model",
			"source_ref": map[string]interface{}{"repo_id": "test/model"},
		}},
		{name: "missing source_ref", args: map[string]any{
			"name":        "my-model",
			"source_type": "SOURCE_TYPE_HUGGINGFACE",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.importModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_importModel_spacesSourceRef(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":        "my-spaces-model",
		"source_type": "SOURCE_TYPE_SPACES_BUCKET",
		"source_ref": map[string]interface{}{
			"bucket": "my-bucket",
			"region": "nyc3",
			"prefix": "models/mistral/",
		},
		"accept_terms_and_conditions": true,
	}}}
	_, err := tool.importModel(context.Background(), req)
	require.Error(t, err, "should fail due to client error, not validation")
}

func TestCustomModelsTool_importModel_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":        "my-model",
		"source_type": "SOURCE_TYPE_HUGGINGFACE",
		"source_ref":  map[string]interface{}{"repo_id": "test/model"},
	}}}
	_, err := tool.importModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_updateMetadata_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{
			"name": "new-name",
		}},
		{name: "empty uuid", args: map[string]any{
			"uuid": "",
			"name": "new-name",
		}},
		{name: "no update fields", args: map[string]any{
			"uuid": "test-uuid",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateMetadata(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_updateMetadata_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "test-uuid",
		"name": "new-name",
	}}}
	_, err := tool.updateMetadata(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_getModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{}},
		{name: "empty uuid", args: map[string]any{"uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_getModel_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "test-uuid",
	}}}
	_, err := tool.getModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_deleteModel_validation(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()

	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing uuid", args: map[string]any{}},
		{name: "empty uuid", args: map[string]any{"uuid": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteModel(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, resp.IsError)
		})
	}
}

func TestCustomModelsTool_deleteModel_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"uuid": "test-uuid",
	}}}
	_, err := tool.deleteModel(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_unifiedSearch_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"query": "llama",
	}}}
	_, err := tool.unifiedSearch(context.Background(), req)
	require.Error(t, err)
}

func TestCustomModelsTool_unifiedSearch_emptyQuery_clientError(t *testing.T) {
	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	_, err := tool.unifiedSearch(context.Background(), req)
	require.Error(t, err, "empty query should still attempt API call and fail on client error")
}

func TestCustomModelMatchesQuery(t *testing.T) {
	tests := []struct {
		name     string
		model    *CustomModel
		query    string
		expected bool
	}{
		{
			name:     "match by name",
			model:    &CustomModel{Name: "my-llama-model"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by description",
			model:    &CustomModel{Name: "some-model", Description: "A fine-tuned Llama variant"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by tag",
			model:    &CustomModel{Name: "some-model", Tags: &CustomModelTags{Tags: []string{"llm", "llama"}}},
			query:    "llama",
			expected: true,
		},
		{
			name:     "match by architecture",
			model:    &CustomModel{Name: "some-model", Architecture: "LlamaForCausalLM"},
			query:    "llama",
			expected: true,
		},
		{
			name:     "no match",
			model:    &CustomModel{Name: "my-gpt-model", Description: "A GPT variant"},
			query:    "llama",
			expected: false,
		},
		{
			name:     "case insensitive",
			model:    &CustomModel{Name: "My-LLAMA-Model"},
			query:    "llama",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := customModelMatchesQuery(tc.model, tc.query)
			require.Equal(t, tc.expected, result)
		})
	}
}
