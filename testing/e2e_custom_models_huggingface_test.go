//go:build integration

package testing

import (
	"context"
	"testing"

	genaicustommodels "mcp-digitalocean/pkg/registry/genai-custom-models"

	"github.com/stretchr/testify/require"
)

func requireGitCommitSHA(t *testing.T, sha string) {
	t.Helper()
	require.Len(t, sha, 40)
	require.Regexp(t, `^[0-9a-f]{40}$`, sha)
}

// TestCustomModelsFetchHuggingFaceCommitSHA calls the live Hugging Face Hub API.
func TestCustomModelsFetchHuggingFaceCommitSHA(t *testing.T) {
	t.Parallel()

	sha, err := genaicustommodels.FetchHuggingFaceCommitSHA(context.Background(), e2eHFRepoID, "")
	require.NoError(t, err)
	requireGitCommitSHA(t, sha)

	sha2, err := genaicustommodels.FetchHuggingFaceCommitSHA(context.Background(), e2eHFRepoID, "")
	require.NoError(t, err)
	require.Equal(t, sha, sha2)
}

// TestCustomModelsResolveHuggingFaceCommitSHA verifies commit resolution before any DigitalOcean import.
func TestCustomModelsResolveHuggingFaceCommitSHA(t *testing.T) {
	t.Parallel()

	ref := &genaicustommodels.CustomModelSourceRef{RepoID: e2eHFRepoID}
	err := genaicustommodels.ResolveHuggingFaceCommitSHA(context.Background(), ref)
	require.NoError(t, err)
	requireGitCommitSHA(t, ref.CommitSHA)
}

// TestCustomModelsFetchHuggingFaceCommitSHA_invalidRepo expects a clear failure from Hugging Face.
func TestCustomModelsFetchHuggingFaceCommitSHA_invalidRepo(t *testing.T) {
	t.Parallel()

	_, err := genaicustommodels.FetchHuggingFaceCommitSHA(context.Background(), "this-org/does-not-exist-xyz-404", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "404")
}
