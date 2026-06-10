package genai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const modelEvalAPIPath = genAIAPIPath + "/model_evaluation"
const modelEvalRunsAPIPath = genAIAPIPath + "/model_evaluation_runs"
const modelEvalPresetsAPIPath = genAIAPIPath + "/model_evaluation_presets"
const modelEvalMetricsAPIPath = genAIAPIPath + "/model_evaluation_metrics"

// ModelEvaluationTool provides model evaluation management tools.
type ModelEvaluationTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewModelEvaluationTool creates a new ModelEvaluationTool instance.
func NewModelEvaluationTool(client func(ctx context.Context) (*godo.Client, error)) *ModelEvaluationTool {
	return &ModelEvaluationTool{
		client: client,
	}
}

// listMetrics lists all available model evaluation metrics.
func (met *ModelEvaluationTool) listMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalMetricsAPIPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListModelEvaluationMetricsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation metrics", err), nil
	}

	type MetricsResponse struct {
		Metrics []*EvaluationMetric `json:"metrics"`
		Count   int                 `json:"count"`
	}

	response := MetricsResponse{
		Metrics: output.Metrics,
		Count:   len(output.Metrics),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listPresets lists all model evaluation presets.
func (met *ModelEvaluationTool) listPresets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalPresetsAPIPath, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListModelEvaluationPresetsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation presets", err), nil
	}

	type PresetsResponse struct {
		Presets []*ModelEvaluationPreset `json:"presets"`
		Count   int                      `json:"count"`
	}

	response := PresetsResponse{
		Presets: output.Presets,
		Count:   len(output.Presets),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getPreset gets a single model evaluation preset by UUID.
func (met *ModelEvaluationTool) getPreset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	presetUUID, ok := req.GetArguments()["eval_preset_uuid"].(string)
	if !ok || presetUUID == "" {
		return mcp.NewToolResultError("eval_preset_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalPresetsAPIPath+"/"+presetUUID, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output GetModelEvaluationPresetOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to get model evaluation preset", err), nil
	}

	jsonData, err := json.MarshalIndent(output.Preset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// createDataset uploads a CSV file and returns presigned URL info for model evaluation.
func (met *ModelEvaluationTool) createDataset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path is required"), nil
	}

	if !isCSVFile(filePath) {
		return mcp.NewToolResultError("file must have .csv extension"), nil
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	fileSize := int64(len(fileData))
	fileName := getFileName(filePath)

	presignedInput := &ModelEvalDatasetPresignedUrlsInput{
		Files: []PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: fileSize,
			},
		},
	}

	presignedReq, err := newGodoRequestWithContext(ctx, client, "POST", modelEvalAPIPath+"/datasets/file_upload_presigned_urls", presignedInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create presigned URL request", err), nil
	}

	var presignedOutput ModelEvalDatasetPresignedUrlsOutput
	resp, err := client.Do(ctx, presignedReq, &presignedOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to create presigned URL", err), nil
	}

	if len(presignedOutput.Uploads) == 0 {
		return mcp.NewToolResultError("no presigned URL returned"), nil
	}

	upload := presignedOutput.Uploads[0]

	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create upload request", err), nil
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "text/csv")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to upload file", err), nil
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))), nil
	}

	type DatasetUploadResponse struct {
		ObjectKey string `json:"object_key"`
		Name      string `json:"name"`
		FileName  string `json:"file_name"`
		FileSize  int64  `json:"file_size"`
	}

	response := DatasetUploadResponse{
		ObjectKey: upload.ObjectKey,
		Name:      name,
		FileName:  fileName,
		FileSize:  fileSize,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// createRun creates a new model evaluation run.
func (met *ModelEvaluationTool) createRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	candidateModelUUID, _ := args["candidate_model_uuid"].(string)
	candidateModelName, _ := args["candidate_model_name"].(string)

	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if candidateModelUUID == "" {
		return mcp.NewToolResultError("candidate_model_uuid is required"), nil
	}
	if candidateModelName == "" {
		return mcp.NewToolResultError("candidate_model_name is required"), nil
	}

	input := &CreateModelEvalRunInput{
		Name:               name,
		CandidateModelUUID: candidateModelUUID,
		CandidateModelName: candidateModelName,
	}

	if presetUUID, ok := args["eval_preset_uuid"].(string); ok && presetUUID != "" {
		input.EvalPresetUUID = &presetUUID
	}

	if datasetUUID, ok := args["dataset_uuid"].(string); ok && datasetUUID != "" {
		input.DatasetUUID = datasetUUID
	}

	if judgeModelUUID, ok := args["judge_model_uuid"].(string); ok && judgeModelUUID != "" {
		input.JudgeModelUUID = judgeModelUUID
	}

	if metricUUIDsRaw, ok := args["metric_uuids"].([]interface{}); ok {
		for _, m := range metricUUIDsRaw {
			if s, ok := m.(string); ok {
				input.MetricUUIDs = append(input.MetricUUIDs, s)
			}
		}
	}

	input.StarMetric = parseStarMetricArg(args)

	if source, ok := args["source"].(string); ok && source != "" {
		input.Source = source
	}

	if saveAsPreset, ok := args["save_as_preset"].(bool); ok {
		input.SaveAsPreset = saveAsPreset
	}

	if presetName, ok := args["preset_name"].(string); ok && presetName != "" {
		input.PresetName = presetName
	}

	if candidateModelSource, ok := args["candidate_model_source"].(string); ok && candidateModelSource != "" {
		input.CandidateModelSource = candidateModelSource
	}

	if inferenceConfigRaw, ok := args["candidate_inference_config"].(map[string]interface{}); ok {
		config := &CandidateInferenceConfig{}
		if maxTokens, ok := inferenceConfigRaw["max_tokens"].(float64); ok {
			v := int(maxTokens)
			config.MaxTokens = &v
		}
		if temp, ok := inferenceConfigRaw["temperature"].(float64); ok {
			config.Temperature = &temp
		}
		if topP, ok := inferenceConfigRaw["top_p"].(float64); ok {
			config.TopP = &topP
		}
		input.CandidateInferenceConfig = config
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "POST", modelEvalRunsAPIPath, input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output CreateModelEvalRunOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to create model evaluation run", err), nil
	}

	type RunCreatedResponse struct {
		EvalRunUUID string `json:"eval_run_uuid"`
		Name        string `json:"name"`
	}

	response := RunCreatedResponse{
		EvalRunUUID: output.EvalRunUUID,
		Name:        name,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// listRuns lists model evaluation runs with optional filters.
func (met *ModelEvaluationTool) listRuns(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	q := url.Values{}
	if presetUUID, ok := args["eval_preset_uuid"].(string); ok && presetUUID != "" {
		q.Set("eval_preset_uuid", presetUUID)
	}
	if status, ok := args["status"].(string); ok && status != "" {
		q.Set("status", status)
	}
	if page, ok := args["page"].(float64); ok {
		q.Set("page", fmt.Sprintf("%d", int(page)))
	}
	if perPage, ok := args["per_page"].(float64); ok {
		q.Set("per_page", fmt.Sprintf("%d", int(perPage)))
	}

	path := modelEvalRunsAPIPath
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ListModelEvaluationRunsOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to list model evaluation runs", err), nil
	}

	type RunsListResponse struct {
		Runs  []*ModelEvaluationRunSummary `json:"runs"`
		Count int                          `json:"count"`
	}

	response := RunsListResponse{
		Runs:  output.Runs,
		Count: len(output.Runs),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getRun gets a single model evaluation run with per-prompt results.
func (met *ModelEvaluationTool) getRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, ok := args["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	q := url.Values{}
	if page, ok := args["page"].(float64); ok {
		q.Set("page", fmt.Sprintf("%d", int(page)))
	}
	if perPage, ok := args["per_page"].(float64); ok {
		q.Set("per_page", fmt.Sprintf("%d", int(perPage)))
	}

	path := modelEvalRunsAPIPath + "/" + runUUID
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", path, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output GetModelEvaluationRunOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to get model evaluation run", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// updateRun updates mutable fields on a model evaluation run (currently name only).
func (met *ModelEvaluationTool) updateRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	runUUID, ok := args["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	input := &UpdateModelEvalRunInput{Name: name}
	apiReq, err := newGodoRequestWithContext(ctx, client, "PATCH", modelEvalRunsAPIPath+"/"+runUUID, input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output UpdateModelEvalRunOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to update model evaluation run", err), nil
	}

	if output.Run == nil {
		return mcp.NewToolResultError("empty response from update model evaluation run"), nil
	}

	jsonData, err := json.MarshalIndent(output.Run, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getResultsDownloadURL gets a presigned download URL for run results.
func (met *ModelEvaluationTool) getResultsDownloadURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runUUID, ok := req.GetArguments()["eval_run_uuid"].(string)
	if !ok || runUUID == "" {
		return mcp.NewToolResultError("eval_run_uuid is required"), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	apiReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalRunsAPIPath+"/"+runUUID+"/results/download_url", nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
	}

	var output ModelEvalResultsDownloadURLOutput
	resp, err := client.Do(ctx, apiReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("failed to get results download URL", err), nil
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// isModelEvalTerminalStatus checks if a model evaluation run status is terminal.
func isModelEvalTerminalStatus(status ModelEvaluationRunStatus) bool {
	switch status {
	case ModelEvalRunStatusSuccessful,
		ModelEvalRunStatusFailed,
		ModelEvalRunStatusCancelled,
		ModelEvalRunStatusPartiallySuccessful:
		return true
	default:
		return false
	}
}

// runWorkflow orchestrates a complete model evaluation: upload dataset, create run, poll for results.
func (met *ModelEvaluationTool) runWorkflow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	datasetFilePath, _ := args["dataset_file_path"].(string)
	runName, _ := args["name"].(string)
	candidateModelUUID, _ := args["candidate_model_uuid"].(string)
	candidateModelName, _ := args["candidate_model_name"].(string)
	judgeModelUUID, _ := args["judge_model_uuid"].(string)

	if datasetFilePath == "" || runName == "" || candidateModelUUID == "" || candidateModelName == "" || judgeModelUUID == "" {
		return mcp.NewToolResultError("dataset_file_path, name, candidate_model_uuid, candidate_model_name, and judge_model_uuid are required"), nil
	}

	timeoutSec := int64(300)
	if t, ok := args["timeout_seconds"].(float64); ok {
		timeoutSec = int64(t)
	}
	pollIntervalSec := int64(5)
	if p, ok := args["poll_interval_seconds"].(float64); ok {
		pollIntervalSec = int64(p)
	}

	// Convert metric UUIDs
	var metricUUIDs []string
	if metricUUIDsRaw, ok := args["metric_uuids"].([]interface{}); ok {
		for _, m := range metricUUIDsRaw {
			if s, ok := m.(string); ok {
				metricUUIDs = append(metricUUIDs, s)
			}
		}
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	startTime := time.Now()

	// Step 1: Validate dataset
	if !isCSVFile(datasetFilePath) {
		return mcp.NewToolResultError("step 1: file must have .csv extension"), nil
	}

	fileData, err := os.ReadFile(datasetFilePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: failed to read dataset file", err), nil
	}

	fileSize := int64(len(fileData))
	fileName := getFileName(datasetFilePath)

	// Step 2: Get presigned URL and upload dataset
	presignedInput := &ModelEvalDatasetPresignedUrlsInput{
		Files: []PresignedUrlFile{
			{
				FileName: fileName,
				FileSize: fileSize,
			},
		},
	}

	presignedReq, err := newGodoRequestWithContext(ctx, client, "POST", modelEvalAPIPath+"/datasets/file_upload_presigned_urls", presignedInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to create presigned URL request", err), nil
	}

	var presignedOutput ModelEvalDatasetPresignedUrlsOutput
	resp, err := client.Do(ctx, presignedReq, &presignedOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 2: failed to create presigned URL", err), nil
	}

	if len(presignedOutput.Uploads) == 0 {
		return mcp.NewToolResultError("step 2: no presigned URL returned"), nil
	}

	upload := presignedOutput.Uploads[0]

	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to create upload request", err), nil
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "text/csv")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to upload file", err), nil
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultError(fmt.Sprintf("step 2: file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))), nil
	}

	// Step 3: List metrics if none provided
	if len(metricUUIDs) == 0 {
		metricsReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalMetricsAPIPath, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 3: failed to create request", err), nil
		}

		var metricsOutput ListModelEvaluationMetricsOutput
		resp, err = client.Do(ctx, metricsReq, &metricsOutput)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 3: failed to list metrics", err), nil
		}

		for _, m := range metricsOutput.Metrics {
			metricUUIDs = append(metricUUIDs, m.MetricUUID)
		}
	}

	// Step 4: Create evaluation run
	runInput := &CreateModelEvalRunInput{
		Name:               runName,
		CandidateModelUUID: candidateModelUUID,
		CandidateModelName: candidateModelName,
		DatasetUUID:        upload.ObjectKey,
		JudgeModelUUID:     judgeModelUUID,
		MetricUUIDs:        metricUUIDs,
		StarMetric:         defaultStarMetric(metricUUIDs),
		Source:             "mcp",
	}

	if inferenceConfigRaw, ok := args["candidate_inference_config"].(map[string]interface{}); ok {
		config := &CandidateInferenceConfig{}
		if maxTokens, ok := inferenceConfigRaw["max_tokens"].(float64); ok {
			v := int(maxTokens)
			config.MaxTokens = &v
		}
		if temp, ok := inferenceConfigRaw["temperature"].(float64); ok {
			config.Temperature = &temp
		}
		if topP, ok := inferenceConfigRaw["top_p"].(float64); ok {
			config.TopP = &topP
		}
		runInput.CandidateInferenceConfig = config
	}

	runReq, err := newGodoRequestWithContext(ctx, client, "POST", modelEvalRunsAPIPath, runInput)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create request", err), nil
	}

	var runOutput CreateModelEvalRunOutput
	resp, err = client.Do(ctx, runReq, &runOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 4: failed to create model evaluation run", err), nil
	}

	evalRunUUID := runOutput.EvalRunUUID

	// Step 5: Poll for completion
	timeout := time.Duration(timeoutSec) * time.Second
	pollInterval := time.Duration(pollIntervalSec) * time.Second
	deadline := time.Now().Add(timeout)

	var finalRun *ModelEvaluationRunDetail
	for {
		if time.Now().After(deadline) {
			return mcp.NewToolResultError("step 5: evaluation polling timed out"), nil
		}

		getReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalRunsAPIPath+"/"+evalRunUUID, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 5: failed to create request", err), nil
		}

		var output GetModelEvaluationRunOutput
		resp, err = client.Do(ctx, getReq, &output)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 5: failed to poll evaluation run", err), nil
		}

		finalRun = output.Run
		if finalRun == nil {
			return mcp.NewToolResultError("step 5: evaluation run missing from API response"), nil
		}

		if isModelEvalTerminalStatus(finalRun.Status) {
			break
		}

		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("workflow cancelled"), nil
		case <-time.After(pollInterval):
		}
	}

	duration := time.Since(startTime).Seconds()

	type WorkflowResponse struct {
		EvalRunUUID     string                   `json:"eval_run_uuid"`
		Status          string                   `json:"status"`
		MetricResults   []map[string]interface{} `json:"metric_results"`
		DurationSeconds float64                  `json:"duration_seconds"`
		ErrorMessage    string                   `json:"error_message,omitempty"`
	}

	metricResults := []map[string]interface{}{}
	if finalRun.StarMetricResult != nil {
		metricResults = append(metricResults, metricResultToMap(finalRun.StarMetricResult))
	}
	for i := range finalRun.RunLevelMetricResults {
		metricResults = append(metricResults, metricResultToMap(&finalRun.RunLevelMetricResults[i]))
	}

	response := WorkflowResponse{
		EvalRunUUID:     evalRunUUID,
		Status:          string(finalRun.Status),
		MetricResults:   metricResults,
		DurationSeconds: duration,
		ErrorMessage:    derefString(finalRun.ErrorDescription),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// Tools returns the list of server tools for model evaluation management.
func (met *ModelEvaluationTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: met.listMetrics,
			Tool: mcp.NewTool(
				"genai-model-eval-list-metrics",
				mcp.WithDescription("List all available model evaluation metrics."),
			),
		},
		{
			Handler: met.listPresets,
			Tool: mcp.NewTool(
				"genai-model-eval-list-presets",
				mcp.WithDescription("List all model evaluation presets. Presets are reusable evaluation configurations containing a dataset, judge model, and metrics."),
			),
		},
		{
			Handler: met.getPreset,
			Tool: mcp.NewTool(
				"genai-model-eval-get-preset",
				mcp.WithDescription("Get a single model evaluation preset by UUID."),
				mcp.WithString("eval_preset_uuid", mcp.Required(), mcp.Description("UUID of the evaluation preset")),
			),
		},
		{
			Handler: met.createDataset,
			Tool: mcp.NewTool(
				"genai-model-eval-create-dataset",
				mcp.WithDescription("Upload a CSV file as a model evaluation dataset. Returns the uploaded object key for use when creating evaluation runs."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the dataset")),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the CSV file to upload")),
			),
		},
		{
			Handler: met.createRun,
			Tool: mcp.NewTool(
				"genai-model-eval-create-run",
				mcp.WithDescription("Create a model evaluation run. Provide either an eval_preset_uuid to use a preset, or provide dataset_uuid, judge_model_uuid, and metric_uuids for inline configuration."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for this evaluation run")),
				mcp.WithString("candidate_model_uuid", mcp.Required(), mcp.Description("UUID of the candidate model to evaluate")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Display name of the candidate model")),
				mcp.WithString("eval_preset_uuid", mcp.Description("UUID of a preset to use (optional; if provided, dataset/judge/metrics come from the preset)")),
				mcp.WithString("dataset_uuid", mcp.Description("UUID of the evaluation dataset (required if not using a preset)")),
				mcp.WithString("judge_model_uuid", mcp.Description("UUID of the judge model that scores responses (required if not using a preset)")),
				mcp.WithObject("metric_uuids", mcp.Description("Array of metric UUIDs to evaluate (required if not using a preset)")),
				mcp.WithObject("star_metric", mcp.Description("Primary success metric: metric_uuid and optional success_threshold_pct")),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithString("source", mcp.Description("Source identifier for this run (e.g., 'mcp')")),
				mcp.WithBoolean("save_as_preset", mcp.Description("Whether to save this configuration as a new preset")),
				mcp.WithString("preset_name", mcp.Description("Name for the new preset (required if save_as_preset is true)")),
				mcp.WithString("candidate_model_source", mcp.Description("Source of the candidate model")),
			),
		},
		{
			Handler: met.listRuns,
			Tool: mcp.NewTool(
				"genai-model-eval-list-runs",
				mcp.WithDescription("List model evaluation runs with optional filters."),
				mcp.WithString("eval_preset_uuid", mcp.Description("Filter by preset UUID")),
				mcp.WithString("status", mcp.Description("Filter by run status (e.g., SUCCESSFUL, FAILED, QUEUED)")),
				mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 1)")),
				mcp.WithNumber("per_page", mcp.Description("Results per page for pagination (default: 20)")),
			),
		},
		{
			Handler: met.getRun,
			Tool: mcp.NewTool(
				"genai-model-eval-get-run",
				mcp.WithDescription("Get the status, details, and per-prompt results of a model evaluation run."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run")),
				mcp.WithNumber("page", mcp.Description("Page number for per-prompt results pagination")),
				mcp.WithNumber("per_page", mcp.Description("Results per page for per-prompt results pagination")),
			),
		},
		{
			Handler: met.updateRun,
			Tool: mcp.NewTool(
				"genai-model-eval-update-run",
				mcp.WithDescription("Update a model evaluation run. Currently only the run name can be changed."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run")),
				mcp.WithString("name", mcp.Required(), mcp.Description("New name for the evaluation run")),
			),
		},
		{
			Handler: met.getResultsDownloadURL,
			Tool: mcp.NewTool(
				"genai-model-eval-get-results-download-url",
				mcp.WithDescription("Get a presigned download URL for the full results of a model evaluation run."),
				mcp.WithString("eval_run_uuid", mcp.Required(), mcp.Description("UUID of the evaluation run")),
			),
		},
		{
			Handler: met.runWorkflow,
			Tool: mcp.NewTool(
				"genai-model-eval-run-workflow",
				mcp.WithDescription("Run a complete model evaluation workflow: upload dataset, create evaluation run, and poll for results. This is a convenience tool that handles all steps automatically."),
				mcp.WithString("dataset_file_path", mcp.Required(), mcp.Description("Path to the CSV evaluation dataset")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the evaluation run")),
				mcp.WithString("candidate_model_uuid", mcp.Required(), mcp.Description("UUID of the candidate model to evaluate")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Display name of the candidate model")),
				mcp.WithString("judge_model_uuid", mcp.Required(), mcp.Description("UUID of the judge model that scores responses")),
				mcp.WithObject("metric_uuids", mcp.Description("Array of metric UUIDs to evaluate (if empty, all available metrics are used)")),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithNumber("timeout_seconds", mcp.Description("Timeout for polling evaluation results in seconds (default: 300)")),
				mcp.WithNumber("poll_interval_seconds", mcp.Description("Interval between status polls in seconds (default: 5)")),
			),
		},
	}
}
