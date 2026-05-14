package genaicustommodels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	sourceCatalog = "catalog"
	sourceCustom  = "custom"
)

// unifiedSearch searches both the model catalog and custom models, returning
// a merged, normalized list. The catalog is searched via GradientAI.SearchModels
// and custom models are fetched from the v2/gen-ai/custom_models endpoint with
// client-side name/description/tag substring matching.
func (cmt *CustomModelsTool) unifiedSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, _ := req.GetArguments()["query"].(string)

	client, err := cmt.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	type catalogResult struct {
		models []*UnifiedModel
		err    error
	}
	type customResult struct {
		models []*UnifiedModel
		err    error
	}

	var wg sync.WaitGroup
	catalogCh := make(chan catalogResult, 1)
	customCh := make(chan customResult, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		models, err := fetchCatalogModels(ctx, client, query)
		catalogCh <- catalogResult{models: models, err: err}
	}()
	go func() {
		defer wg.Done()
		models, err := fetchCustomModels(ctx, client, query)
		customCh <- customResult{models: models, err: err}
	}()

	wg.Wait()
	close(catalogCh)
	close(customCh)

	cr := <-catalogCh
	cu := <-customCh

	var catalogModels, customModels []*UnifiedModel
	var errors []string

	if cr.err != nil {
		errors = append(errors, fmt.Sprintf("catalog: %s", cr.err))
	} else {
		catalogModels = cr.models
	}

	if cu.err != nil {
		errors = append(errors, fmt.Sprintf("custom: %s", cu.err))
	} else {
		customModels = cu.models
	}

	if len(errors) == 2 {
		return mcp.NewToolResultError(fmt.Sprintf("both sources failed: %s", strings.Join(errors, "; "))), nil
	}

	merged := make([]*UnifiedModel, 0, len(catalogModels)+len(customModels))
	merged = append(merged, catalogModels...)
	merged = append(merged, customModels...)

	resp := UnifiedSearchResponse{
		Query:   query,
		Results: merged,
	}
	resp.Counts.Catalog = len(catalogModels)
	resp.Counts.Custom = len(customModels)
	resp.Counts.Total = len(merged)

	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	result := mcp.NewToolResultText(string(jsonData))
	if len(errors) == 1 {
		result = mcp.NewToolResultText(fmt.Sprintf("partial results (one source failed: %s)\n\n%s", errors[0], string(jsonData)))
	}

	return result, nil
}

// fetchCatalogModels searches the model catalog and fetches metadata for each
// matching UUID, converting results into UnifiedModel.
func fetchCatalogModels(ctx context.Context, client *godo.Client, query string) ([]*UnifiedModel, error) {
	uuids, _, err := client.GradientAI.SearchModels(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog: %w", err)
	}

	type result struct {
		model *UnifiedModel
		err   error
	}

	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)
	results := make([]result, len(uuids))

	var wg sync.WaitGroup
	for i, uuid := range uuids {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			model, _, getErr := client.GradientAI.GetModelByUUID(ctx, id)
			if getErr != nil || model == nil {
				results[idx] = result{err: getErr}
				return
			}

			var inputMod, outputMod []string
			if model.Modalities != nil {
				inputMod = model.Modalities.Input
				outputMod = model.Modalities.Output
			}

			results[idx] = result{model: &UnifiedModel{
				UUID:             model.Uuid,
				Name:             model.Name,
				Description:      model.Description,
				Source:           sourceCatalog,
				Provider:         model.Provider,
				Type:             model.Type,
				ContextWindow:    model.ContextWindow,
				Capabilities:     model.Capabilities,
				InputModalities:  inputMod,
				OutputModalities: outputMod,
			}}
		}(i, uuid)
	}
	wg.Wait()

	models := make([]*UnifiedModel, 0, len(results))
	for _, r := range results {
		if r.model != nil {
			models = append(models, r.model)
		}
	}
	return models, nil
}

// fetchCustomModels lists all custom models and applies client-side substring
// filtering on name, description, and tags when a query is provided.
func fetchCustomModels(ctx context.Context, client *godo.Client, query string) ([]*UnifiedModel, error) {
	apiReq, err := client.NewRequest(ctx, http.MethodGet, customModelsAPIPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, apiReq.Method, apiReq.URL.String(), apiReq.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}
	httpReq.Header = apiReq.Header.Clone()

	var output ListCustomModelsOutput
	resp, err := client.Do(ctx, httpReq, &output)
	if err != nil || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to list custom models: %w", err)
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	models := make([]*UnifiedModel, 0, len(output.Models))
	for _, cm := range output.Models {
		if queryLower != "" && !customModelMatchesQuery(cm, queryLower) {
			continue
		}
		models = append(models, &UnifiedModel{
			UUID:             cm.UUID,
			Name:             cm.Name,
			Description:      cm.Description,
			Source:           sourceCustom,
			Status:           string(cm.Status),
			Architecture:     cm.Architecture,
			InputModalities:  cm.InputModalities,
			OutputModalities: cm.OutputModalities,
		})
	}
	return models, nil
}

func customModelMatchesQuery(cm *CustomModel, queryLower string) bool {
	if strings.Contains(strings.ToLower(cm.Name), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(cm.Description), queryLower) {
		return true
	}
	if cm.Tags != nil {
		for _, tag := range cm.Tags.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				return true
			}
		}
	}
	if strings.Contains(strings.ToLower(cm.Architecture), queryLower) {
		return true
	}
	return false
}
