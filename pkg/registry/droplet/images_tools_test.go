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
func newTestTool(t *testing.T) (*ImageTool, *MockImagesService) {
	ctrl := gomock.NewController(t)
	m := NewMockImagesService(ctrl)
	return NewImageTool(func(context.Context) (*godo.Client, error) {
		return &godo.Client{Images: m}, nil
	}), m
}

func TestImageTool_listImages(t *testing.T) {
	images := []godo.Image{{ID: 1, Name: "Ubuntu"}, {ID: 2, Name: "Backup"}}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImagesService)
		wantErr bool
	}{
		{
			name: "List all (default)",
			args: map[string]any{"Page": 1.0, "PerPage": 10.0},
			setup: func(m *MockImagesService) {
				m.EXPECT().List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 10}).Return(images, nil, nil)
			},
		},
		{
			name: "List distributions",
			args: map[string]any{"Type": "distribution"},
			setup: func(m *MockImagesService) {
				m.EXPECT().ListDistribution(gomock.Any(), gomock.Any()).Return(images, nil, nil)
			},
		},
		{
			name: "List applications",
			args: map[string]any{"Type": "application"},
			setup: func(m *MockImagesService) {
				m.EXPECT().ListApplication(gomock.Any(), gomock.Any()).Return(images, nil, nil)
			},
		},
		{
			name: "List user images",
			args: map[string]any{"Type": "user"},
			setup: func(m *MockImagesService) {
				m.EXPECT().ListUser(gomock.Any(), gomock.Any()).Return(images, nil, nil)
			},
		},
		{
			name: "API Error",
			args: map[string]any{},
			setup: func(m *MockImagesService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}

			res, err := tool.listImages(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})

			if tc.wantErr {
				require.True(t, res.IsError)
			} else {
				require.NoError(t, err)
				require.False(t, res.IsError)
				require.NotEmpty(t, res.Content)
			}
		})
	}
}

func TestImageTool_getImageByID(t *testing.T) {
	image := &godo.Image{ID: 123, Name: "test-image"}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImagesService)
		wantErr bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ID": 123.0},
			setup: func(m *MockImagesService) {
				m.EXPECT().GetByID(gomock.Any(), 123).Return(image, nil, nil)
			},
		},
		{
			name: "API Error",
			args: map[string]any{"ID": 456.0},
			setup: func(m *MockImagesService) {
				m.EXPECT().GetByID(gomock.Any(), 456).Return(nil, nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name:    "Missing ID",
			args:    map[string]any{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}

			res, err := tool.getImageByID(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})

			if tc.wantErr {
				require.True(t, res.IsError)
			} else {
				require.NoError(t, err)
				var out map[string]any
				json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out)
				assert.Equal(t, image.Name, out["name"])
			}
		})
	}
}

func TestImageTool_createImage(t *testing.T) {
	image := &godo.Image{ID: 123, Name: "custom-image"}
	baseArgs := map[string]any{
		"Name":         "custom-image",
		"Url":          "http://example.com/image.iso",
		"Region":       "nyc3",
		"Distribution": "Ubuntu",
		"Description":  "A custom image",
		"Tags":         []any{"custom"},
	}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImagesService)
		wantErr bool
	}{
		{
			name: "Successful create",
			args: baseArgs,
			setup: func(m *MockImagesService) {
				m.EXPECT().Create(gomock.Any(), &godo.CustomImageCreateRequest{
					Name:         "custom-image",
					Url:          "http://example.com/image.iso",
					Region:       "nyc3",
					Distribution: "Ubuntu",
					Description:  "A custom image",
					Tags:         []string{"custom"},
				}).Return(image, nil, nil)
			},
		},
		{name: "Missing Name", args: map[string]any{"Url": "u", "Region": "r"}, wantErr: true},
		{name: "Missing Url", args: map[string]any{"Name": "n", "Region": "r"}, wantErr: true},
		{name: "Missing Region", args: map[string]any{"Name": "n", "Url": "u"}, wantErr: true},
		{
			name: "API Error",
			args: baseArgs,
			setup: func(m *MockImagesService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}
			res, _ := tool.createImage(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})
			require.Equal(t, tc.wantErr, res.IsError)
		})
	}
}

func TestImageTool_updateImage(t *testing.T) {
	image := &godo.Image{ID: 123, Name: "new-name"}

	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImagesService)
		wantErr bool
	}{
		{
			name: "Successful update",
			args: map[string]any{"ID": 123.0, "Name": "new-name"},
			setup: func(m *MockImagesService) {
				m.EXPECT().Update(gomock.Any(), 123, &godo.ImageUpdateRequest{Name: "new-name"}).Return(image, nil, nil)
			},
		},
		{name: "Missing Name", args: map[string]any{"ID": 123.0}, wantErr: true},
		{name: "Missing ID", args: map[string]any{"Name": "new"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}
			res, _ := tool.updateImage(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})
			require.Equal(t, tc.wantErr, res.IsError)
		})
	}
}

func TestImageTool_deleteImage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		setup   func(*MockImagesService)
		wantErr bool
	}{
		{
			name: "Successful delete",
			args: map[string]any{"ID": 123.0},
			setup: func(m *MockImagesService) {
				m.EXPECT().Delete(gomock.Any(), 123).Return(nil, nil)
			},
		},
		{
			name: "API Error",
			args: map[string]any{"ID": 456.0},
			setup: func(m *MockImagesService) {
				m.EXPECT().Delete(gomock.Any(), 456).Return(nil, errors.New("error"))
			},
			wantErr: true,
		},
		{name: "Missing ID", args: map[string]any{}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, m := newTestTool(t)
			if tc.setup != nil {
				tc.setup(m)
			}
			res, _ := tool.deleteImage(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.args},
			})
			require.Equal(t, tc.wantErr, res.IsError)
			if !tc.wantErr {
				assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "deleted successfully")
			}
		})
	}
}
