package genai

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/digitalocean/godo"
)

// ModelEvalDatasetResult is returned after uploading and registering a model evaluation dataset.
type ModelEvalDatasetResult struct {
	EvaluationDatasetUUID string `json:"evaluation_dataset_uuid"`
	DatasetUUID           string `json:"dataset_uuid"`
	ObjectKey             string `json:"object_key"`
	Name                  string `json:"name"`
	FileName              string `json:"file_name"`
	FileSize              int64  `json:"file_size"`
}

// validateModelEvaluationDataset checks that a CSV has the required input column for model evaluation.
func validateModelEvaluationDataset(filePath string) error {
	if !isCSVFile(filePath) {
		return fmt.Errorf("file must have .csv extension")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	hasInput := false
	for _, col := range header {
		if col == "input" {
			hasInput = true
			break
		}
	}
	if !hasInput {
		return fmt.Errorf("CSV must contain an 'input' column")
	}

	rowNum := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}
		rowNum++
		if len(row) == 0 {
			return fmt.Errorf("row %d: empty row", rowNum)
		}
	}

	return nil
}

// uploadAndRegisterModelEvaluationDataset presigns, uploads to Spaces, and registers the dataset record.
func uploadAndRegisterModelEvaluationDataset(
	ctx context.Context,
	client *godo.Client,
	name string,
	fileData []byte,
	fileName string,
) (*ModelEvalDatasetResult, error) {
	fileSize := int64(len(fileData))

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
		return nil, fmt.Errorf("failed to create presigned URL request: %w", err)
	}

	var presignedOutput ModelEvalDatasetPresignedUrlsOutput
	resp, err := client.Do(ctx, presignedReq, &presignedOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to create presigned URL: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create presigned URL: status %d", resp.StatusCode)
	}

	if len(presignedOutput.Uploads) == 0 {
		return nil, fmt.Errorf("no presigned URL returned")
	}

	upload := presignedOutput.Uploads[0]

	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", upload.PresignedURL, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "text/csv")

	httpResp, err := presignedUploadHTTPClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("file upload failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	datasetInput := &CreateEvaluationDatasetInput{
		Name:        name,
		DatasetType: EvaluationDatasetTypeModel,
		FileUploadDataSource: FileUploadDataSource{
			OriginalFileName: fileName,
			StoredObjectKey:  upload.ObjectKey,
			SizeInBytes:      fileSize,
		},
	}

	datasetReq, err := newGodoRequestWithContext(ctx, client, "POST", genAIAPIPath+"/evaluation_datasets", datasetInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataset request: %w", err)
	}

	var datasetOutput CreateEvaluationDatasetOutput
	resp, err = client.Do(ctx, datasetReq, &datasetOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to register evaluation dataset: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to register evaluation dataset: status %d", resp.StatusCode)
	}

	if datasetOutput.EvaluationDatasetUUID == "" {
		return nil, fmt.Errorf("evaluation dataset registration returned empty UUID")
	}

	return &ModelEvalDatasetResult{
		EvaluationDatasetUUID: datasetOutput.EvaluationDatasetUUID,
		DatasetUUID:           datasetOutput.EvaluationDatasetUUID,
		ObjectKey:             upload.ObjectKey,
		Name:                  name,
		FileName:              fileName,
		FileSize:              fileSize,
	}, nil
}
