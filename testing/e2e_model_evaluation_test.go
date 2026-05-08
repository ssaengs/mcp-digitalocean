//go:build integration

package testing

import (
	"os"
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

// TestModelEvalListRunsWithPagination tests pagination parameters on list-runs.
func TestModelEvalListRunsWithPagination(t *testing.T) {
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

	out := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"page":     float64(1),
		"per_page": float64(5),
	})

	require.GreaterOrEqual(t, out.Count, 0)
	require.LessOrEqual(t, out.Count, 5, "per_page=5 should return at most 5 runs")
	require.Len(t, out.Runs, out.Count)
	t.Logf("listed %d model evaluation run(s) with pagination", out.Count)
}

// TestModelEvalGetRun gets a single run using a UUID from list-runs.
// Skipped if no runs exist.
func TestModelEvalGetRun(t *testing.T) {
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

	listOut := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"page":     float64(1),
		"per_page": float64(1),
	})

	if listOut.Count == 0 {
		t.Skip("no model evaluation runs exist, skipping get-run test")
	}

	runUUID := listOut.Runs[0].EvalRunUUID
	require.NotEmpty(t, runUUID)

	type runDetail struct {
		EvalRunUUID        string `json:"eval_run_uuid"`
		Name               string `json:"name"`
		Status             string `json:"status"`
		CandidateModelUUID string `json:"candidate_model_uuid"`
		CandidateModelName string `json:"candidate_model_name"`
	}
	type getRunResponse struct {
		Run *runDetail `json:"run"`
	}

	out := callTool[getRunResponse](t, "genai-model-eval-get-run", map[string]any{
		"eval_run_uuid": runUUID,
	})

	require.NotNil(t, out.Run, "run detail should not be nil")
	require.Equal(t, runUUID, out.Run.EvalRunUUID)
	require.NotEmpty(t, out.Run.Name)
	require.NotEmpty(t, out.Run.Status)
	t.Logf("got run %s: name=%s status=%s", out.Run.EvalRunUUID, out.Run.Name, out.Run.Status)
}

// TestModelEvalGetResultsDownloadURL gets a download URL for a completed run.
// Skipped if no successful runs exist.
func TestModelEvalGetResultsDownloadURL(t *testing.T) {
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

	listOut := callTool[listRunsResponse](t, "genai-model-eval-list-runs", map[string]any{
		"status": "MODEL_EVALUATION_RUN_SUCCESSFUL",
	})

	if listOut.Count == 0 {
		t.Skip("no successful model evaluation runs exist, skipping download URL test")
	}

	runUUID := listOut.Runs[0].EvalRunUUID
	require.NotEmpty(t, runUUID)

	type downloadURLResponse struct {
		DownloadURL string `json:"download_url"`
	}

	out := callTool[downloadURLResponse](t, "genai-model-eval-get-results-download-url", map[string]any{
		"eval_run_uuid": runUUID,
	})

	require.NotEmpty(t, out.DownloadURL, "download URL should not be empty")
	t.Logf("got download URL for run %s", runUUID)
}

// TestModelEvalCreateDataset uploads a test CSV dataset.
// Requires MODEL_EVAL_TEST_DATASET_PATH env var pointing to a CSV file,
// or creates a minimal test file.
func TestModelEvalCreateDataset(t *testing.T) {
	t.Parallel()

	datasetPath := os.Getenv("MODEL_EVAL_TEST_DATASET_PATH")
	if datasetPath == "" {
		tmpFile, err := os.CreateTemp("", "model-eval-test-*.csv")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("query,expected_output\n\"{\\\"role\\\": \\\"user\\\", \\\"content\\\": \\\"What is 2+2?\\\"}\",\"4\"\n")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())
		datasetPath = tmpFile.Name()
	}

	type datasetResponse struct {
		ObjectKey string `json:"object_key"`
		Name      string `json:"name"`
		FileName  string `json:"file_name"`
		FileSize  int64  `json:"file_size"`
	}

	out := callTool[datasetResponse](t, "genai-model-eval-create-dataset", map[string]any{
		"name":      "e2e-test-dataset",
		"file_path": datasetPath,
	})

	require.NotEmpty(t, out.ObjectKey, "object_key should not be empty")
	require.Equal(t, "e2e-test-dataset", out.Name)
	require.Greater(t, out.FileSize, int64(0), "file size should be positive")
	t.Logf("uploaded dataset: object_key=%s file_size=%d", out.ObjectKey, out.FileSize)
}
