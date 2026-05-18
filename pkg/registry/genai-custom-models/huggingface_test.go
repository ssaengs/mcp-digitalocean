package genaicustommodels

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestEnsureHuggingFaceCommitSHA_usesProvidedSHA(t *testing.T) {
	ref := &CustomModelSourceRef{
		RepoID:    "org/model",
		CommitSHA: "already-set",
	}
	err := ensureHuggingFaceCommitSHA(context.Background(), ref)
	require.NoError(t, err)
	require.Equal(t, "already-set", ref.CommitSHA)
}

func TestEnsureHuggingFaceCommitSHA_fetchesWhenMissing(t *testing.T) {
	oldFetch := fetchHuggingFaceCommitSHA
	fetchHuggingFaceCommitSHA = func(ctx context.Context, repoID, hfToken string) (string, error) {
		require.Equal(t, "org/model", repoID)
		require.Empty(t, hfToken)
		return "fetched-sha", nil
	}
	t.Cleanup(func() { fetchHuggingFaceCommitSHA = oldFetch })

	ref := &CustomModelSourceRef{RepoID: "org/model"}
	err := ensureHuggingFaceCommitSHA(context.Background(), ref)
	require.NoError(t, err)
	require.Equal(t, "fetched-sha", ref.CommitSHA)
}

func TestEnsureHuggingFaceCommitSHA_requiresRepoID(t *testing.T) {
	ref := &CustomModelSourceRef{}
	err := ensureHuggingFaceCommitSHA(context.Background(), ref)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repo_id")
}

func TestDefaultFetchHuggingFaceCommitSHA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/models/org/model", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]string{"sha": "token-sha"})
	}))
	t.Cleanup(srv.Close)

	oldBase := huggingFaceModelsAPIBase
	huggingFaceModelsAPIBase = srv.URL + "/api/models"
	t.Cleanup(func() { huggingFaceModelsAPIBase = oldBase })

	sha, err := defaultFetchHuggingFaceCommitSHA(context.Background(), "org/model", "secret")
	require.NoError(t, err)
	require.Equal(t, "token-sha", sha)
}

func TestImportModel_hfFetchFailsBeforeClient(t *testing.T) {
	oldFetch := fetchHuggingFaceCommitSHA
	fetchHuggingFaceCommitSHA = func(ctx context.Context, repoID, hfToken string) (string, error) {
		return "", context.Canceled
	}
	t.Cleanup(func() { fetchHuggingFaceCommitSHA = oldFetch })

	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":                        "my-model",
		"source_type":                 "SOURCE_TYPE_HUGGINGFACE",
		"source_ref":                  map[string]interface{}{"repo_id": "test/model"},
		"accept_terms_and_conditions": true,
	}}}
	resp, err := tool.importModel(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError)
}

func TestImportModel_hfResolvesCommitBeforeClient(t *testing.T) {
	oldFetch := fetchHuggingFaceCommitSHA
	fetchHuggingFaceCommitSHA = func(ctx context.Context, repoID, hfToken string) (string, error) {
		return "resolved-sha", nil
	}
	t.Cleanup(func() { fetchHuggingFaceCommitSHA = oldFetch })

	tool := setupCustomModelsToolWithFailingClient()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"name":                        "my-model",
		"source_type":                 "SOURCE_TYPE_HUGGINGFACE",
		"source_ref":                  map[string]interface{}{"repo_id": "test/model"},
		"accept_terms_and_conditions": true,
	}}}
	_, err := tool.importModel(context.Background(), req)
	require.Error(t, err, "should fail at DigitalOcean client after commit_sha is resolved")
}
