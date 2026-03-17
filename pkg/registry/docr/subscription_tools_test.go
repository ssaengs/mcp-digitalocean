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

func setupSubscriptionToolWithMock(mockRegistries godo.RegistriesService) *SubscriptionTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Registries: mockRegistries,
		}, nil
	}
	return NewSubscriptionTool(client)
}

func TestSubscriptionTool_getSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testSubscription := &godo.RegistrySubscription{
		Tier: &godo.RegistrySubscriptionTier{Name: "Basic"},
	}

	tests := []struct {
		name        string
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name: "api error",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetSubscription(gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().GetSubscription(gomock.Any()).Return(testSubscription, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockRegistriesService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupSubscriptionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.getSubscription(context.Background(), req)
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

func TestSubscriptionTool_updateSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testSubscription := &godo.RegistrySubscription{
		Tier: &godo.RegistrySubscriptionTier{Name: "Professional"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockRegistriesService)
		expectError bool
	}{
		{
			name:        "missing TierSlug",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"TierSlug": "professional"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().UpdateSubscription(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"TierSlug": "professional"},
			mockSetup: func(m *MockRegistriesService) {
				m.EXPECT().UpdateSubscription(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *godo.RegistrySubscriptionUpdateRequest) (*godo.RegistrySubscription, *godo.Response, error) {
						require.Equal(t, "professional", req.TierSlug)
						return testSubscription, nil, nil
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
			tool := setupSubscriptionToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateSubscription(context.Background(), req)
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
