//go:build integration

package testing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelEvalListMetrics calls genai-model-eval-list-metrics against the live GenAI API.
func TestModelEvalListMetrics(t *testing.T) {
	t.Parallel()

	type metric struct {
		MetricUUID string `json:"metric_uuid"`
		MetricName string `json:"metric_name"`
	}
	type listMetricsResponse struct {
		Metrics []metric `json:"metrics"`
		Count   int      `json:"count"`
	}

	out := callTool[listMetricsResponse](t, "genai-model-eval-list-metrics", map[string]any{})

	require.Greater(t, out.Count, 0, "expected at least one model evaluation metric from API")
	require.Len(t, out.Metrics, out.Count)
	require.NotEmpty(t, out.Metrics[0].MetricUUID)
	require.NotEmpty(t, out.Metrics[0].MetricName)
	t.Logf("listed %d model evaluation metric(s)", out.Count)
}

// TestModelEvalListPresets calls genai-model-eval-list-presets against the live GenAI API.
func TestModelEvalListPresets(t *testing.T) {
	t.Parallel()

	type preset struct {
		EvalPresetUUID string `json:"eval_preset_uuid"`
		Name           string `json:"name"`
	}
	type listPresetsResponse struct {
		Presets []preset `json:"presets"`
		Count   int      `json:"count"`
	}

	out := callTool[listPresetsResponse](t, "genai-model-eval-list-presets", map[string]any{})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Presets, out.Count)
	t.Logf("listed %d model evaluation preset(s)", out.Count)
}

// TestModelEvalListRuns calls genai-model-eval-list-runs against the live GenAI API.
func TestModelEvalListRuns(t *testing.T) {
	t.Parallel()

	type runSummary struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
		Status      string `json:"status"`
	}
	type listRunsResponse struct {
		Runs  []runSummary `json:"runs"`
		Count int          `json:"count"`
	}

	out := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.Runs, out.Count)
	t.Logf("listed %d model evaluation run(s)", out.Count)
}
