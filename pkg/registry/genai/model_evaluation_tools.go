package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
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

// createDataset uploads a CSV to Spaces and registers it as a model evaluation dataset.
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

	if err := validateModelEvaluationDataset(filePath); err != nil {
		return mcp.NewToolResultErrorFromErr("dataset validation failed", err), nil
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to read file", err), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	result, err := uploadAndRegisterModelEvaluationDataset(ctx, client, name, fileData, getFileName(filePath))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create model evaluation dataset", err), nil
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// resolveEvalModelsOnly resolves candidate and judge models before chat-based user consent.
func (met *ModelEvaluationTool) resolveEvalModelsOnly(
	ctx context.Context,
	args map[string]any,
	requireJudge bool,
) (*modelEvalRunModels, *mcp.CallToolResult, error) {
	candidateModelName := strings.TrimSpace(stringArg(args, "candidate_model_name"))
	if candidateModelName == "" {
		return nil, mcp.NewToolResultError(modelEvalCandidateNameRequiredMsg), nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	models, err := listAllEvalModels(ctx, client)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr("failed to list models for evaluation run resolution", err), nil
	}

	resolved, unresolved, err := resolveEvalModelsForRun(
		ctx,
		client,
		strings.TrimSpace(stringArg(args, "candidate_model_uuid")),
		candidateModelName,
		strings.TrimSpace(stringArg(args, "judge_model_uuid")),
		strings.TrimSpace(stringArg(args, "judge_model_name")),
		requireJudge,
		models,
	)
	if err != nil {
		return nil, mcp.NewToolResultError(err.Error()), nil
	}
	if unresolved != nil {
		result, err := modelEvalUserActionResult(unresolved)
		return nil, result, err
	}

	hydrated, err := hydrateResolvedEvalModelsFromAPI(ctx, client, resolved)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr("failed to fetch catalog UUIDs for candidate/judge models", err), nil
	}
	if err := validateModelEvalResolvedUUIDs(hydrated, requireJudge); err != nil {
		return nil, mcp.NewToolResultError(err.Error()), nil
	}

	return hydrated, nil, nil
}

// createRun creates a new model evaluation run.
func (met *ModelEvaluationTool) createRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	presetUUID, _ := args["eval_preset_uuid"].(string)
	requireJudge := presetUUID == ""

	runCfg, err := parseModelEvalRunConfig(args, requireJudge)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resolved, selectionResult, err := met.resolveEvalModelsOnly(ctx, args, requireJudge)
	if selectionResult != nil || err != nil {
		return selectionResult, err
	}

	if consentResult, ok := checkModelEvalUserMessage(args, resolved, runCfg); !ok {
		return consentResult, nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	input := &CreateModelEvalRunInput{
		Name:               name,
		CandidateModelUUID: resolved.Candidate.UUID,
		CandidateModelName: resolved.Candidate.APIName,
	}

	if presetUUID != "" {
		input.EvalPresetUUID = &presetUUID
	}

	if datasetUUID, ok := args["dataset_uuid"].(string); ok && datasetUUID != "" {
		input.DatasetUUID = datasetUUID
	}

	if resolved.Judge != nil {
		input.JudgeModelUUID = resolved.Judge.UUID
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

	if datasetFilePath == "" || runName == "" {
		return mcp.NewToolResultError("dataset_file_path and name are required"), nil
	}
	if strings.TrimSpace(stringArg(args, "candidate_model_name")) == "" {
		return mcp.NewToolResultError(modelEvalCandidateNameRequiredMsg), nil
	}
	if strings.TrimSpace(stringArg(args, "judge_model_name")) == "" {
		return mcp.NewToolResultError(modelEvalJudgeNameRequiredMsg), nil
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

	resolved, selectionResult, err := met.resolveEvalModelsOnly(ctx, args, true)
	if selectionResult != nil || err != nil {
		return selectionResult, err
	}

	workflowCfg := &modelEvalRunConfig{
		RunName:         runName,
		DatasetFilePath: datasetFilePath,
		MetricUUIDs:     append([]string(nil), metricUUIDs...),
		StarMetric:      parseStarMetricArg(args),
	}
	if workflowCfg.StarMetric == nil && len(metricUUIDs) > 0 {
		workflowCfg.StarMetric = defaultStarMetric(metricUUIDs)
	}

	if consentResult, ok := checkModelEvalUserMessage(args, resolved, workflowCfg); !ok {
		return consentResult, nil
	}

	client, err := met.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	startTime := time.Now()

	if err := validateModelEvaluationDataset(datasetFilePath); err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: dataset validation failed", err), nil
	}

	fileData, err := os.ReadFile(datasetFilePath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 1: failed to read dataset file", err), nil
	}

	fileName := getFileName(datasetFilePath)
	datasetName := runName + "-dataset"

	// Steps 2–4: presign, upload to Spaces, register dataset record.
	datasetResult, err := uploadAndRegisterModelEvaluationDataset(ctx, client, datasetName, fileData, fileName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("step 2: failed to upload and register dataset", err), nil
	}

	// Step 5: List metrics if none provided
	if len(metricUUIDs) == 0 {
		metricsReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalMetricsAPIPath, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 5: failed to create request", err), nil
		}

		var metricsOutput ListModelEvaluationMetricsOutput
		resp, err := client.Do(ctx, metricsReq, &metricsOutput)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 5: failed to list metrics", err), nil
		}

		for _, m := range metricsOutput.Metrics {
			metricUUIDs = append(metricUUIDs, m.MetricUUID)
		}
	}

	// Step 6: Create evaluation run
	runInput := &CreateModelEvalRunInput{
		Name:               runName,
		CandidateModelUUID: resolved.Candidate.UUID,
		CandidateModelName: resolved.Candidate.APIName,
		DatasetUUID:        datasetResult.EvaluationDatasetUUID,
		JudgeModelUUID:     resolved.Judge.UUID,
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
		return mcp.NewToolResultErrorFromErr("step 6: failed to create request", err), nil
	}

	var runOutput CreateModelEvalRunOutput
	resp, err := client.Do(ctx, runReq, &runOutput)
	if err != nil || resp.StatusCode >= 400 {
		return mcp.NewToolResultErrorFromErr("step 6: failed to create model evaluation run", err), nil
	}

	evalRunUUID := runOutput.EvalRunUUID

	// Step 7: Poll for completion
	timeout := time.Duration(timeoutSec) * time.Second
	pollInterval := time.Duration(pollIntervalSec) * time.Second
	deadline := time.Now().Add(timeout)

	var finalRun *ModelEvaluationRunDetail
	for {
		if time.Now().After(deadline) {
			return mcp.NewToolResultError("step 7: evaluation polling timed out"), nil
		}

		getReq, err := newGodoRequestWithContext(ctx, client, "GET", modelEvalRunsAPIPath+"/"+evalRunUUID, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("step 7: failed to create request", err), nil
		}

		var output GetModelEvaluationRunOutput
		resp, err = client.Do(ctx, getReq, &output)
		if err != nil || resp.StatusCode >= 400 {
			return mcp.NewToolResultErrorFromErr("step 7: failed to poll evaluation run", err), nil
		}

		finalRun = output.Run
		if finalRun == nil {
			return mcp.NewToolResultError("step 7: evaluation run missing from API response"), nil
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
				mcp.WithDescription("Upload and register a model evaluation dataset (presign → Spaces upload → database record). CSV must include an 'input' column; 'ground_truth' is optional. Returns evaluation_dataset_uuid for use with genai-model-eval-create-run."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the dataset")),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the CSV file to upload")),
			),
		},
		{
			Handler: met.createRun,
			Tool: mcp.NewTool(
				"genai-model-eval-create-run",
				mcp.WithDescription(genaiModelEvalCreateRunToolDescription),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for this evaluation run")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Exact candidate model name the user provided or confirmed (character-for-character, whitespace trimmed). Partial names return nearest matches only.")),
				mcp.WithString("candidate_model_uuid", mcp.Description("Exact full candidate model UUID (8-4-4-4-12 hex). Optional if name is exact; partial uuids return matches only.")),
				mcp.WithString("eval_preset_uuid", mcp.Description("UUID of a preset to use (optional; if provided, dataset/judge/metrics come from the preset)")),
				mcp.WithString("dataset_uuid", mcp.Description("evaluation_dataset_uuid from genai-model-eval-create-dataset (required if not using a preset)")),
				mcp.WithString("judge_model_name", mcp.Description("Exact judge model name (required for inline configuration without a preset). Partial names return nearest matches only.")),
				mcp.WithString("judge_model_uuid", mcp.Description("Exact full judge model UUID. Optional if judge_model_name is exact; partial uuids return matches only.")),
				mcp.WithObject("metric_uuids", mcp.Description("Array of metric UUIDs to evaluate (required if not using a preset)")),
				mcp.WithObject("star_metric", mcp.Description("Primary success metric: metric_uuid and optional success_threshold_pct")),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithString("source", mcp.Description("Source identifier for this run (e.g., 'mcp')")),
				mcp.WithBoolean("save_as_preset", mcp.Description("Whether to save this configuration as a new preset")),
				mcp.WithString("preset_name", mcp.Description("Name for the new preset (required if save_as_preset is true)")),
				mcp.WithString("candidate_model_source", mcp.Description("Source of the candidate model")),
				mcp.WithString("user_message", mcp.Description(genaiModelEvalUserMessageDescription)),
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
				mcp.WithDescription(genaiModelEvalWorkflowToolDescription),
				mcp.WithString("dataset_file_path", mcp.Required(), mcp.Description("Path to the CSV evaluation dataset")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name for the evaluation run")),
				mcp.WithString("candidate_model_name", mcp.Required(), mcp.Description("Exact candidate model name. Partial names return nearest matches only.")),
				mcp.WithString("candidate_model_uuid", mcp.Description("Exact full candidate model UUID. Optional if name is exact.")),
				mcp.WithString("judge_model_name", mcp.Required(), mcp.Description("Exact judge model name. Partial names return nearest matches only.")),
				mcp.WithString("judge_model_uuid", mcp.Description("Exact full judge model UUID. Optional if judge_model_name is exact.")),
				mcp.WithObject("metric_uuids", mcp.Description("Array of metric UUIDs to evaluate (if empty, all available metrics are used)")),
				mcp.WithObject("candidate_inference_config", mcp.Description("Inference parameters: max_tokens (int), temperature (float), top_p (float)")),
				mcp.WithNumber("timeout_seconds", mcp.Description("Timeout for polling evaluation results in seconds (default: 300)")),
				mcp.WithNumber("poll_interval_seconds", mcp.Description("Interval between status polls in seconds (default: 5)")),
				mcp.WithString("user_message", mcp.Description(genaiModelEvalUserMessageDescription)),
			),
		},
	}
}
