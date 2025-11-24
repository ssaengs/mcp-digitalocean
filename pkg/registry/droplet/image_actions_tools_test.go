package droplet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Helper to initialize tool and mock
func newTestActionTool(t *testing.T) (*ImageActionsTool, *MockImageActionsService) {
	ctrl := gomock.NewController(t)
	m := NewMockImageActionsService(ctrl)
	return NewImageActionsTool(func(context.Context) (*godo.Client, error) {
		return &godo.Client{ImageActions: m}, nil
	}), m
}

func TestImageActionsTool_transferImage(t *testing.T) {
	action := &godo.Action{ID: 1, Status: "in-progress", Type: "transfer"}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImageActionsService)
		wantErr bool
	}{
		{
			name: "Successful transfer",
			args: map[string]any{"ID": 123.0, "Region": "nyc3"},
			setup: func(m *MockImageActionsService) {
				req := &godo.ActionRequest{"type": "transfer", "region": "nyc3"}
				m.EXPECT().Transfer(gomock.Any(), 123, req).Return(action, nil, nil)
			},
		},
		{name: "Missing ID", args: map[string]any{"Region": "nyc3"}, wantErr: true},
		{name: "Missing Region", args: map[string]any{"ID": 123.0}, wantErr: true},
		{
			name: "API Error",
			args: map[string]any{"ID": 456.0, "Region": "ams3"},
			setup: func(m *MockImageActionsService) {
				m.EXPECT().Transfer(gomock.Any(), 456, gomock.Any()).Return(nil, nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestActionTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}

			res, err := tool.transferImage(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})

			require.Equal(t, tc.wantErr, res.IsError)
			if !tc.wantErr {
				require.NoError(t, err)
				var out godo.Action
				json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out)
				assert.Equal(t, action.ID, out.ID)
			}
		})
	}
}

func TestImageActionsTool_convertImageToSnapshot(t *testing.T) {
	action := &godo.Action{ID: 2, Status: "completed", Type: "convert"}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImageActionsService)
		wantErr bool
	}{
		{
			name: "Successful convert",
			args: map[string]any{"ID": 123.0},
			setup: func(m *MockImageActionsService) {
				m.EXPECT().Convert(gomock.Any(), 123).Return(action, nil, nil)
			},
		},
		{name: "Missing ID", args: map[string]any{}, wantErr: true},
		{
			name: "API Error",
			args: map[string]any{"ID": 456.0},
			setup: func(m *MockImageActionsService) {
				m.EXPECT().Convert(gomock.Any(), 456).Return(nil, nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestActionTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}

			res, _ := tool.convertImageToSnapshot(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})

			require.Equal(t, tc.wantErr, res.IsError)
		})
	}
}

func TestImageActionsTool_getImageAction(t *testing.T) {
	action := &godo.Action{ID: 999, Status: "completed"}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImageActionsService)
		wantErr bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ImageID": 123.0, "ActionID": 999.0},
			setup: func(m *MockImageActionsService) {
				m.EXPECT().Get(gomock.Any(), 123, 999).Return(action, nil, nil)
			},
		},
		{name: "Missing ImageID", args: map[string]any{"ActionID": 999.0}, wantErr: true},
		{name: "Missing ActionID", args: map[string]any{"ImageID": 123.0}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestActionTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}

			res, _ := tool.getImageAction(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})

			require.Equal(t, tc.wantErr, res.IsError)
		})
	}
}
