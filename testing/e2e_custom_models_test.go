//go:build integration

package testing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCustomModelsListModels calls genai-custom-models-list against the live GenAI API.
func TestCustomModelsListModels(t *testing.T) {
	t.Parallel()

	type customModel struct {
		UUID   string `json:"uuid"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	type listResponse struct {
		Models       []customModel `json:"models"`
		Count        int           `json:"count"`
		MaxThreshold int           `json:"max_threshold"`
	}

	out := callTool[listResponse](t, "genai-custom-models-list", map[string]any{})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Models, out.Count)
	t.Logf("listed %d custom model(s)", out.Count)
}

// TestCustomModelsListModelsWithPagination tests pagination on custom models list.
func TestCustomModelsListModelsWithPagination(t *testing.T) {
	t.Parallel()

	type customModel struct {
		UUID   string `json:"uuid"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	type listResponse struct {
		Models []customModel `json:"models"`
		Count  int           `json:"count"`
	}

	out := callTool[listResponse](t, "genai-custom-models-list", map[string]any{
		"page":     float64(1),
		"per_page": float64(5),
	})

	require.GreaterOrEqual(t, out.Count, 0)
	require.LessOrEqual(t, out.Count, 5, "per_page=5 should return at most 5 models")
	require.Len(t, out.Models, out.Count)
	t.Logf("listed %d custom model(s) with pagination", out.Count)
}

// TestCustomModelsListModelsWithStatusFilter tests status filtering on custom models list.
func TestCustomModelsListModelsWithStatusFilter(t *testing.T) {
	t.Parallel()

	type customModel struct {
		UUID   string `json:"uuid"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	type listResponse struct {
		Models []customModel `json:"models"`
		Count  int           `json:"count"`
	}

	out := callTool[listResponse](t, "genai-custom-models-list", map[string]any{
		"status": "STATUS_READY",
	})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Models, out.Count)
	for _, m := range out.Models {
		require.Equal(t, "STATUS_READY", m.Status, "all models should have STATUS_READY")
	}
	t.Logf("listed %d custom model(s) with STATUS_READY filter", out.Count)
}
