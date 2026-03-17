package docr

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupRepositoryToolWithMock(mockRegistries godo.RegistriesService) *RepositoryTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Registries: mockRegistries,
		}, nil
	}
	return NewRepositoryTool(client)
}

func TestRepositoryTool_listRepositories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testRepos := []*godo.RepositoryV2{{Name: "repo1"}, {Name: "repo2"}}
	testMeta := &godo.Meta{Total: 2, Page: 1, Pages: 1}

	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*MockRegistriesService)
		expectError    bool
		expectContains []string
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "Page": float64(1), "PerPage": float64(20)},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoriesV2(gomock.Any(), "my-registry", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Page": float64(1), "PerPage": float64(10)},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoriesV2(gomock.Any(), "my-registry", gomock.Any()).DoAndReturn(
					func(_ context.Context, name string, opts *godo.TokenListOptions) ([]*godo.RepositoryV2, *godo.Response, error) {
						require.Equal(t, 1, opts.Page)
						require.Equal(t, 10, opts.PerPage)
						return testRepos, &godo.Response{Meta: testMeta}, nil
					},
				)
			},
			expectContains: []string{`"meta"`, `"total": 2`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRepositoryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listRepositories(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			for _, s := range tc.expectContains {
				require.Contains(t, textContent.Text, s)
			}
		})
	}
}

func TestRepositoryTool_listRepositoryTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testTags := []*godo.RepositoryTag{{Tag: "latest"}, {Tag: "v1.0"}}
	testTagsMeta := &godo.Meta{Total: 2, Page: 1, Pages: 1}

	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*MockRegistriesService)
		expectError    bool
		expectContains []string
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{"Repository": "my-repo"},
			expectError: true,
		},
		{
			name:        "missing Repository",
			args:        map[string]any{"RegistryName": "my-registry"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoryTags(gomock.Any(), "my-registry", "my-repo", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo", "Page": float64(1), "PerPage": float64(5)},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoryTags(gomock.Any(), "my-registry", "my-repo", gomock.Any()).DoAndReturn(
					func(_ context.Context, regName, repoName string, opts *godo.ListOptions) ([]*godo.RepositoryTag, *godo.Response, error) {
						require.Equal(t, 1, opts.Page)
						require.Equal(t, 5, opts.PerPage)
						return testTags, &godo.Response{Meta: testTagsMeta}, nil
					},
				)
			},
			expectContains: []string{`"meta"`, `"total": 2`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRepositoryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listRepositoryTags(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			for _, s := range tc.expectContains {
				require.Contains(t, textContent.Text, s)
			}
		})
	}
}

func TestRepositoryTool_deleteTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{"Repository": "my-repo", "Tag": "latest"},
			expectError: true,
		},
		{
			name:        "missing Repository",
			args:        map[string]any{"RegistryName": "my-registry", "Tag": "latest"},
			expectError: true,
		},
		{
			name:        "missing Tag",
			args:        map[string]any{"RegistryName": "my-registry", "Repository": "my-repo"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo", "Tag": "latest"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DeleteTag(gomock.Any(), "my-registry", "my-repo", "latest").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo", "Tag": "latest"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DeleteTag(gomock.Any(), "my-registry", "my-repo", "latest").Return(nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRepositoryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteTag(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestRepositoryTool_listRepositoryManifests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testManifests := []*godo.RepositoryManifest{{Digest: "sha256:abc123"}}
	testManifestsMeta := &godo.Meta{Total: 1, Page: 1, Pages: 1}

	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*MockRegistriesService)
		expectError    bool
		expectContains []string
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{"Repository": "my-repo"},
			expectError: true,
		},
		{
			name:        "missing Repository",
			args:        map[string]any{"RegistryName": "my-registry"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoryManifests(gomock.Any(), "my-registry", "my-repo", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ListRepositoryManifests(gomock.Any(), "my-registry", "my-repo", gomock.Any()).Return(testManifests, &godo.Response{Meta: testManifestsMeta}, nil)
			},
			expectContains: []string{`"meta"`, `"total": 1`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRepositoryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listRepositoryManifests(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			for _, s := range tc.expectContains {
				require.Contains(t, textContent.Text, s)
			}
		})
	}
}

func TestRepositoryTool_deleteManifest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{"Repository": "my-repo", "Digest": "sha256:abc"},
			expectError: true,
		},
		{
			name:        "missing Repository",
			args:        map[string]any{"RegistryName": "my-registry", "Digest": "sha256:abc"},
			expectError: true,
		},
		{
			name:        "missing Digest",
			args:        map[string]any{"RegistryName": "my-registry", "Repository": "my-repo"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo", "Digest": "sha256:abc"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DeleteManifest(gomock.Any(), "my-registry", "my-repo", "sha256:abc").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "Repository": "my-repo", "Digest": "sha256:abc"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DeleteManifest(gomock.Any(), "my-registry", "my-repo", "sha256:abc").Return(nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRepositoryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteManifest(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}
