package genai

import "time"

// ModelEvaluationRunStatus represents the status of a model evaluation run.
type ModelEvaluationRunStatus string

const (
	ModelEvalRunStatusUnspecified         ModelEvaluationRunStatus = "MODEL_EVALUATION_RUN_STATUS_UNSPECIFIED"
	ModelEvalRunStatusQueued              ModelEvaluationRunStatus = "QUEUED"
	ModelEvalRunStatusRunningDataset      ModelEvaluationRunStatus = "RUNNING_DATASET"
	ModelEvalRunStatusEvaluatingResults   ModelEvaluationRunStatus = "EVALUATING_RESULTS"
	ModelEvalRunStatusCancelling          ModelEvaluationRunStatus = "CANCELLING"
	ModelEvalRunStatusCancelled           ModelEvaluationRunStatus = "CANCELLED"
	ModelEvalRunStatusSuccessful          ModelEvaluationRunStatus = "SUCCESSFUL"
	ModelEvalRunStatusPartiallySuccessful ModelEvaluationRunStatus = "PARTIALLY_SUCCESSFUL"
	ModelEvalRunStatusFailed              ModelEvaluationRunStatus = "FAILED"
)

// ModelEvaluationPreset represents a reusable model evaluation configuration.
type ModelEvaluationPreset struct {
	EvalPresetUUID string              `json:"eval_preset_uuid"`
	Name           string              `json:"name"`
	DatasetUUID    string              `json:"dataset_uuid"`
	DatasetName    string              `json:"dataset_name"`
	JudgeModelUUID string              `json:"judge_model_uuid"`
	JudgeModelName string              `json:"judge_model_name"`
	Metrics        []*EvaluationMetric `json:"metrics,omitempty"`
	StarMetric     *StarMetric         `json:"star_metric,omitempty"`
	CreatedAt      *time.Time          `json:"created_at,omitempty"`
}

// ListModelEvaluationPresetsOutput is the response from listing presets.
type ListModelEvaluationPresetsOutput struct {
	Presets []*ModelEvaluationPreset `json:"presets"`
}

// GetModelEvaluationPresetOutput is the response from getting a single preset.
type GetModelEvaluationPresetOutput struct {
	Preset *ModelEvaluationPreset `json:"preset"`
}

// CandidateInferenceConfig holds inference parameters for the candidate model.
type CandidateInferenceConfig struct {
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

// CreateModelEvalRunInput is the request body for creating a model evaluation run.
type CreateModelEvalRunInput struct {
	Name                     string                    `json:"name"`
	EvalPresetUUID           *string                   `json:"eval_preset_uuid,omitempty"`
	CandidateModelUUID       string                    `json:"candidate_model_uuid"`
	CandidateModelName       string                    `json:"candidate_model_name"`
	CandidateInferenceConfig *CandidateInferenceConfig `json:"candidate_inference_config,omitempty"`
	DatasetUUID              string                    `json:"dataset_uuid,omitempty"`
	JudgeModelUUID           string                    `json:"judge_model_uuid,omitempty"`
	MetricUUIDs              []string                  `json:"metric_uuids,omitempty"`
	StarMetric               *StarMetric               `json:"star_metric,omitempty"`
	Source                   string                    `json:"source,omitempty"`
	SaveAsPreset             bool                      `json:"save_as_preset,omitempty"`
	PresetName               string                    `json:"preset_name,omitempty"`
	CandidateModelSource     string                    `json:"candidate_model_source,omitempty"`
}

// CreateModelEvalRunOutput is the response from creating a model evaluation run.
type CreateModelEvalRunOutput struct {
	EvalRunUUID string `json:"eval_run_uuid"`
}

// ModelEvaluationRunSummary is a summary of a model evaluation run (used in list responses).
type ModelEvaluationRunSummary struct {
	EvalRunUUID          string                   `json:"eval_run_uuid"`
	Name                 string                   `json:"name"`
	Status               ModelEvaluationRunStatus `json:"status"`
	CandidateModelUUID   string                   `json:"candidate_model_uuid"`
	CandidateModelName   string                   `json:"candidate_model_name"`
	JudgeModelUUID       string                   `json:"judge_model_uuid,omitempty"`
	JudgeModelName       string                   `json:"judge_model_name,omitempty"`
	EvalPresetUUID       string                   `json:"eval_preset_uuid,omitempty"`
	StarMetricResult     *EvaluationMetricResult  `json:"star_metric_result,omitempty"`
	PassStatus           *bool                    `json:"pass_status,omitempty"`
	StartedAt            *time.Time               `json:"started_at,omitempty"`
	FinishedAt           *time.Time               `json:"finished_at,omitempty"`
	CreatedAt            *time.Time               `json:"created_at,omitempty"`
	ErrorDescription     *string                  `json:"error_description,omitempty"`
	CandidateModelSource string                   `json:"candidate_model_source,omitempty"`
}

// PaginationLinks holds pagination links from the API.
type PaginationLinks struct {
	Pages *PaginationPages `json:"pages,omitempty"`
}

// PaginationPages holds the page URLs.
type PaginationPages struct {
	First string `json:"first,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Last  string `json:"last,omitempty"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Total int `json:"total"`
}

// ListModelEvaluationRunsOutput is the response from listing model evaluation runs.
type ListModelEvaluationRunsOutput struct {
	Runs  []*ModelEvaluationRunSummary `json:"runs"`
	Links *PaginationLinks             `json:"links,omitempty"`
	Meta  *PaginationMeta              `json:"meta,omitempty"`
}

// ModelEvaluationResult holds per-prompt evaluation results.
type ModelEvaluationResult struct {
	Prompt           string                   `json:"prompt"`
	Response         string                   `json:"response"`
	GroundTruth      *string                  `json:"ground_truth,omitempty"`
	MetricResults    []EvaluationMetricResult `json:"metric_results,omitempty"`
	ErrorDescription *string                  `json:"error_description,omitempty"`
}

// ModelEvaluationRunDetail is the full detail of a model evaluation run.
type ModelEvaluationRunDetail struct {
	EvalRunUUID              string                    `json:"eval_run_uuid"`
	Name                     string                    `json:"name"`
	Status                   ModelEvaluationRunStatus  `json:"status"`
	CandidateModelUUID       string                    `json:"candidate_model_uuid"`
	CandidateModelName       string                    `json:"candidate_model_name"`
	CandidateInferenceConfig *CandidateInferenceConfig `json:"candidate_inference_config,omitempty"`
	JudgeModelUUID           string                    `json:"judge_model_uuid,omitempty"`
	JudgeModelName           string                    `json:"judge_model_name,omitempty"`
	DatasetUUID              string                    `json:"dataset_uuid,omitempty"`
	EvalPresetUUID           string                    `json:"eval_preset_uuid,omitempty"`
	MetricUUIDs              []string                  `json:"metric_uuids,omitempty"`
	StarMetric               *StarMetric               `json:"star_metric,omitempty"`
	StarMetricResult         *EvaluationMetricResult   `json:"star_metric_result,omitempty"`
	RunLevelMetricResults    []EvaluationMetricResult  `json:"run_level_metric_results,omitempty"`
	PassStatus               *bool                     `json:"pass_status,omitempty"`
	StartedAt                *time.Time                `json:"started_at,omitempty"`
	FinishedAt               *time.Time                `json:"finished_at,omitempty"`
	CreatedAt                *time.Time                `json:"created_at,omitempty"`
	ErrorDescription         *string                   `json:"error_description,omitempty"`
	CandidateModelSource     string                    `json:"candidate_model_source,omitempty"`
}

// GetModelEvaluationRunOutput is the response from getting a single model evaluation run.
type GetModelEvaluationRunOutput struct {
	Run     *ModelEvaluationRunDetail `json:"run"`
	Results []*ModelEvaluationResult  `json:"results,omitempty"`
	Links   *PaginationLinks          `json:"links,omitempty"`
	Meta    *PaginationMeta           `json:"meta,omitempty"`
}

// ModelEvalResultsDownloadURLOutput is the response with a presigned download URL for results.
type ModelEvalResultsDownloadURLOutput struct {
	DownloadURL string     `json:"download_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ModelEvalDatasetPresignedUrlsInput is the request for model evaluation dataset presigned URLs.
type ModelEvalDatasetPresignedUrlsInput struct {
	Files []PresignedUrlFile `json:"files"`
}

// ModelEvalDatasetPresignedUrlsOutput is the response with presigned upload URLs.
type ModelEvalDatasetPresignedUrlsOutput struct {
	RequestID string                     `json:"request_id"`
	Uploads   []FilePresignedUrlResponse `json:"uploads"`
}

// ListModelEvaluationMetricsOutput is the response from listing model evaluation metrics.
type ListModelEvaluationMetricsOutput struct {
	Metrics []*EvaluationMetric `json:"metrics,omitempty"`
}
