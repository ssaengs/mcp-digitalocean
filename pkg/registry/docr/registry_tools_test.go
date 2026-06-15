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

func setupRegistryToolWithMock(mockRegistries godo.RegistriesService) *RegistryTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Registries: mockRegistries,
		}, nil
	}
	return NewRegistryTool(client)
}

func TestRegistryTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testRegistry := &godo.Registry{Name: "my-registry", Region: "nyc3"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Get(gomock.Any(), "my-registry").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Get(gomock.Any(), "my-registry").Return(testRegistry, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.get(context.Background(), req)
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
			require.Contains(t, textContent.Text, "my-registry")
		})
	}
}

func TestRegistryTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testRegistries := []*godo.Registry{{Name: "reg1"}, {Name: "reg2"}}

	tests := []struct {
		name        string
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name: "api error",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().List(gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().List(gomock.Any()).Return(testRegistries, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.list(context.Background(), req)
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

func TestRegistryTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testRegistry := &godo.Registry{Name: "my-registry"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing Name",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"Name": "my-registry", "SubscriptionTierSlug": "basic", "Region": "nyc3"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Name": "my-registry", "SubscriptionTierSlug": "basic", "Region": "nyc3"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *godo.RegistryCreateRequest) (*godo.Registry, *godo.Response, error) {
						require.Equal(t, "my-registry", req.Name)
						require.Equal(t, "basic", req.SubscriptionTierSlug)
						require.Equal(t, "nyc3", req.Region)
						return testRegistry, nil, nil
					},
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.create(context.Background(), req)
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

func TestRegistryTool_delete(t *testing.T) {
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
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Delete(gomock.Any(), "my-registry").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().Delete(gomock.Any(), "my-registry").Return(nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.delete(context.Background(), req)
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

func TestRegistryTool_dockerCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testCreds := &godo.DockerCredentials{DockerConfigJSON: []byte(`{"auths":{}}`)}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing RegistryName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"RegistryName": "my-registry"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DockerCredentials(gomock.Any(), "my-registry", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"RegistryName": "my-registry", "ReadWrite": true, "ExpirySeconds": float64(3600)},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().DockerCredentials(gomock.Any(), "my-registry", gomock.Any()).DoAndReturn(
					func(_ context.Context, name string, req *godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, *godo.Response, error) {
						require.True(t, req.ReadWrite)
						require.NotNil(t, req.ExpirySeconds)
						require.Equal(t, 3600, *req.ExpirySeconds)
						return testCreds, nil, nil
					},
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.dockerCredentials(context.Background(), req)
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

func TestRegistryTool_getOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testOptions := &godo.RegistryOptions{AvailableRegions: []string{"nyc3", "sfo3"}}

	tests := []struct {
		name        string
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name: "api error",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetOptions(gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetOptions(gomock.Any()).Return(testOptions, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.getOptions(context.Background(), req)
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

func TestRegistryTool_validateName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing Name",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"Name": "taken-name"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ValidateName(gomock.Any(), gomock.Any()).Return(nil, errors.New("name already taken"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Name": "available-name"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().ValidateName(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *godo.RegistryValidateNameRequest) (*godo.Response, error) {
						require.Equal(t, "available-name", req.Name)
						return nil, nil
					},
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupRegistryToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.validateName(context.Background(), req)
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
