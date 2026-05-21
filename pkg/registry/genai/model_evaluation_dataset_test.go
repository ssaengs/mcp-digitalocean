package genai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateModelEvaluationDataset(t *testing.T) {
	dir := t.TempDir()

	validPath := filepath.Join(dir, "valid.csv")
	require.NoError(t, os.WriteFile(validPath, []byte("input,ground_truth\nWhat is 2+2?,4\n"), 0o600))

	noInputPath := filepath.Join(dir, "no_input.csv")
	require.NoError(t, os.WriteFile(noInputPath, []byte("query,answer\nfoo,bar\n"), 0o600))

	require.NoError(t, validateModelEvaluationDataset(validPath))

	err := validateModelEvaluationDataset(noInputPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "input")

	err = validateModelEvaluationDataset(filepath.Join(dir, "missing.csv"))
	require.Error(t, err)

	err = validateModelEvaluationDataset(filepath.Join(dir, "bad.json"))
	require.Error(t, err)
	require.Contains(t, err.Error(), ".csv")
}

func TestCreateEvaluationDatasetInput_modelType(t *testing.T) {
	input := CreateEvaluationDatasetInput{
		Name:        "test",
		DatasetType: EvaluationDatasetTypeModel,
		FileUploadDataSource: FileUploadDataSource{
			OriginalFileName: "data.csv",
			StoredObjectKey:  "datasets/abc.csv",
			SizeInBytes:      100,
		},
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)
	require.Contains(t, string(data), "EVALUATION_DATASET_TYPE_MODEL")
	require.Contains(t, string(data), "file_upload_dataset")
}
