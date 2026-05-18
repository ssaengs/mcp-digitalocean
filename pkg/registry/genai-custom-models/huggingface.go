package genaicustommodels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var huggingFaceModelsAPIBase = "https://huggingface.co/api/models"

// fetchHuggingFaceCommitSHA resolves the default-branch commit for a Hugging Face repo.
// Overridden in unit tests.
var fetchHuggingFaceCommitSHA = defaultFetchHuggingFaceCommitSHA

// FetchHuggingFaceCommitSHA returns the default-branch Git commit SHA for a Hugging Face model repo.
func FetchHuggingFaceCommitSHA(ctx context.Context, repoID, hfToken string) (string, error) {
	return fetchHuggingFaceCommitSHA(ctx, repoID, hfToken)
}

// ResolveHuggingFaceCommitSHA sets sourceRef.CommitSHA from Hugging Face Hub when it is empty.
func ResolveHuggingFaceCommitSHA(ctx context.Context, sourceRef *CustomModelSourceRef) error {
	return ensureHuggingFaceCommitSHA(ctx, sourceRef)
}

func defaultFetchHuggingFaceCommitSHA(ctx context.Context, repoID, hfToken string) (string, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return "", fmt.Errorf("repo_id is required")
	}

	apiURL, err := url.JoinPath(huggingFaceModelsAPIBase, repoID)
	if err != nil {
		return "", fmt.Errorf("invalid repo_id: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	if hfToken != "" {
		req.Header.Set("Authorization", "Bearer "+hfToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("hugging face API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read hugging face API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hugging face API returned %d for repo %q", resp.StatusCode, repoID)
	}

	var info struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("parse hugging face API response: %w", err)
	}
	if info.SHA == "" {
		return "", fmt.Errorf("hugging face API returned no sha for repo %q", repoID)
	}
	return info.SHA, nil
}

func ensureHuggingFaceCommitSHA(ctx context.Context, sourceRef *CustomModelSourceRef) error {
	if strings.TrimSpace(sourceRef.CommitSHA) != "" {
		return nil
	}
	if strings.TrimSpace(sourceRef.RepoID) == "" {
		return fmt.Errorf("source_ref.repo_id is required for Hugging Face imports")
	}
	sha, err := fetchHuggingFaceCommitSHA(ctx, sourceRef.RepoID, sourceRef.HFToken)
	if err != nil {
		return err
	}
	sourceRef.CommitSHA = sha
	return nil
}
