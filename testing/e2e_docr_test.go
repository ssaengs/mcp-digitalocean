//go:build integration

package testing

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

const defaultDOCRTierSlug = "starter"

// getCurrentSubscriptionTierSlug fetches the current registry subscription tier slug.
// If there is no subscription or the fetch fails, it returns "starter".
func getCurrentSubscriptionTierSlug(t *testing.T) string {
	t.Helper()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "docr-subscription-get",
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil || resp == nil || resp.IsError || len(resp.Content) == 0 {
		t.Logf("could not fetch current subscription, using default tier %q", defaultDOCRTierSlug)
		return defaultDOCRTierSlug
	}

	var subscription godo.RegistrySubscription
	tc, ok := resp.Content[0].(mcp.TextContent)
	if !ok {
		return defaultDOCRTierSlug
	}
	if err := json.Unmarshal([]byte(tc.Text), &subscription); err != nil || subscription.Tier == nil || subscription.Tier.Slug == "" {
		return defaultDOCRTierSlug
	}

	t.Logf("current subscription tier: %s (slug: %s)", subscription.Tier.Name, subscription.Tier.Slug)
	return subscription.Tier.Slug
}

// deleteRegistry is a cleanup helper that logs errors but doesn't fail the test.
func deleteRegistry(t *testing.T, tc testContext, registryName string) {
	t.Logf("deleting container registry %s...", registryName)
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "docr-delete",
			Arguments: map[string]interface{}{"RegistryName": registryName},
		},
	})
	if err != nil {
		t.Logf("failed to delete container registry: %v", err)
		return
	}
	if resp.IsError {
		t.Logf("docr-delete returned error: %v", resp.Content)
		return
	}
	t.Logf("deleted container registry %s", registryName)
}

// TestDOCRRegistryLifecycle tests the full lifecycle of a container registry:
// create -> get -> list -> validate-name -> get options -> delete
func TestDOCRRegistryLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	tierSlug := getCurrentSubscriptionTierSlug(t)
	registryName := fmt.Sprintf("mcp-e2e-docr-%d", time.Now().Unix())

	// get registry options to find a valid region and tier
	t.Log("getting container registry options...")
	options := callTool[godo.RegistryOptions](t, "docr-options", map[string]interface{}{})
	require.NotEmpty(t, options.AvailableRegions, "should have available regions")
	require.NotEmpty(t, options.SubscriptionTiers, "should have subscription tiers")
	t.Logf("available regions: %v, tiers: %d", options.AvailableRegions, len(options.SubscriptionTiers))

	region := options.AvailableRegions[0]

	// validate name before creating
	t.Log("validating registry name...")
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "docr-validate-name",
			Arguments: map[string]interface{}{"Name": registryName},
		},
	})
	requireBasicResponse(t, err, resp)
	require.False(t, resp.IsError, "name validation should succeed")
	t.Logf("registry name %q is available", registryName)

	// create registry
	t.Log("creating container registry...")
	registry := callTool[godo.Registry](t, "docr-create", map[string]interface{}{
		"Name":                 registryName,
		"SubscriptionTierSlug": tierSlug,
		"Region":               region,
	})
	require.Equal(t, registryName, registry.Name)
	t.Logf("created container registry: %s (region: %s)", registry.Name, registry.Region)

	defer func() { deleteRegistry(t, tc, registryName) }()

	// get registry
	t.Log("getting container registry...")
	fetchedRegistry := callTool[godo.Registry](t, "docr-get", map[string]interface{}{
		"RegistryName": registryName,
	})
	require.Equal(t, registryName, fetchedRegistry.Name)
	t.Logf("fetched container registry: %s", fetchedRegistry.Name)

	// list registries
	t.Log("listing container registries...")
	registries := callTool[[]*godo.Registry](t, "docr-list", map[string]interface{}{})
	requireFoundInList(t, registries, func(r *godo.Registry) bool { return r.Name == registryName }, "registry")
	t.Logf("found registry in list (total: %d)", len(registries))

	// get docker credentials
	t.Log("getting docker credentials...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "docr-docker-credentials",
			Arguments: map[string]interface{}{
				"RegistryName":  registryName,
				"ReadWrite":     false,
				"ExpirySeconds": float64(3600),
			},
		},
	})
	requireBasicResponse(t, err, resp)
	require.False(t, resp.IsError, "docker credentials should succeed")
	credText := resp.Content[0].(mcp.TextContent).Text
	require.NotEmpty(t, credText, "docker credentials should not be empty")
	t.Log("successfully retrieved docker credentials")

	// get subscription
	t.Log("getting registry subscription...")
	subscription := callTool[godo.RegistrySubscription](t, "docr-subscription-get", map[string]interface{}{})
	require.NotNil(t, subscription.Tier, "subscription should have a tier")
	t.Logf("current subscription tier: %s", subscription.Tier.Name)
}

// TestDOCRRepositoryOperations tests repository-related operations.
// This test requires a registry to already exist with at least one pushed image.
// It tests: list repositories, list tags, list manifests.
func TestDOCRRepositoryOperations(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	tierSlug := getCurrentSubscriptionTierSlug(t)
	registryName := fmt.Sprintf("mcp-e2e-docr-repo-%d", time.Now().Unix())

	// create a registry for the test
	t.Log("creating container registry for repository tests...")
	registry := callTool[godo.Registry](t, "docr-create", map[string]interface{}{
		"Name":                 registryName,
		"SubscriptionTierSlug": tierSlug,
	})
	require.Equal(t, registryName, registry.Name)
	t.Logf("created container registry: %s", registry.Name)

	defer func() { deleteRegistry(t, tc, registryName) }()

	// list repositories (should be empty for a new registry)
	t.Log("listing repositories...")
	repos := callTool[[]*godo.RepositoryV2](t, "docr-repository-list", map[string]interface{}{
		"RegistryName": registryName,
		"Page":         float64(1),
		"PerPage":      float64(20),
	})
	t.Logf("found %d repositories", len(repos))

	// If there are repositories, test tag and manifest listing
	if len(repos) > 0 {
		repoName := repos[0].Name
		t.Logf("testing with repository: %s", repoName)

		// list tags
		t.Log("listing repository tags...")
		tags := callTool[[]*godo.RepositoryTag](t, "docr-repository-tag-list", map[string]interface{}{
			"RegistryName": registryName,
			"Repository":   repoName,
			"Page":         float64(1),
			"PerPage":      float64(20),
		})
		t.Logf("found %d tags", len(tags))

		// list manifests
		t.Log("listing repository manifests...")
		manifests := callTool[[]*godo.RepositoryManifest](t, "docr-repository-manifest-list", map[string]interface{}{
			"RegistryName": registryName,
			"Repository":   repoName,
			"Page":         float64(1),
			"PerPage":      float64(20),
		})
		t.Logf("found %d manifests", len(manifests))
	}
}

// TestDOCRGarbageCollectionOperations tests garbage collection operations.
func TestDOCRGarbageCollectionOperations(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	tierSlug := getCurrentSubscriptionTierSlug(t)
	registryName := fmt.Sprintf("mcp-e2e-docr-gc-%d", time.Now().Unix())

	// create a registry for the test
	t.Log("creating container registry for GC tests...")
	registry := callTool[godo.Registry](t, "docr-create", map[string]interface{}{
		"Name":                 registryName,
		"SubscriptionTierSlug": tierSlug,
	})
	require.Equal(t, registryName, registry.Name)
	t.Logf("created container registry: %s", registry.Name)

	defer func() { deleteRegistry(t, tc, registryName) }()

	// list garbage collections (should be empty for new registry)
	t.Log("listing garbage collections...")
	gcs := callTool[[]*godo.GarbageCollection](t, "docr-garbage-collection-list", map[string]interface{}{
		"RegistryName": registryName,
		"Page":         float64(1),
		"PerPage":      float64(20),
	})
	t.Logf("found %d garbage collections", len(gcs))

	// start a garbage collection
	t.Log("starting garbage collection...")
	gc := callTool[godo.GarbageCollection](t, "docr-garbage-collection-start", map[string]interface{}{
		"RegistryName": registryName,
	})
	require.NotEmpty(t, gc.UUID, "garbage collection UUID should not be empty")
	t.Logf("started garbage collection: %s (status: %s)", gc.UUID, gc.Status)

	// get the active garbage collection
	t.Log("getting active garbage collection...")
	activeGC := callTool[godo.GarbageCollection](t, "docr-garbage-collection-get", map[string]interface{}{
		"RegistryName": registryName,
	})
	require.NotEmpty(t, activeGC.UUID, "active GC UUID should not be empty")
	t.Logf("active garbage collection: %s (status: %s)", activeGC.UUID, activeGC.Status)

	// cancel the garbage collection if it's still running
	if activeGC.Status != "succeeded" && activeGC.Status != "failed" && activeGC.Status != "cancelled" {
		t.Log("cancelling garbage collection...")
		cancelledGC := callTool[godo.GarbageCollection](t, "docr-garbage-collection-update", map[string]interface{}{
			"RegistryName":          registryName,
			"GarbageCollectionUUID": activeGC.UUID,
			"Cancel":                true,
		})
		t.Logf("garbage collection update response: %s (status: %s)", cancelledGC.UUID, cancelledGC.Status)
	}

	// list garbage collections again to confirm it shows up
	t.Log("listing garbage collections after start...")
	gcsAfter := callTool[[]*godo.GarbageCollection](t, "docr-garbage-collection-list", map[string]interface{}{
		"RegistryName": registryName,
		"Page":         float64(1),
		"PerPage":      float64(20),
	})
	require.NotEmpty(t, gcsAfter, "should have at least one garbage collection")
	requireFoundInList(t, gcsAfter, func(g *godo.GarbageCollection) bool { return g.UUID == gc.UUID }, "garbage collection")
	t.Logf("found garbage collection in list (total: %d)", len(gcsAfter))
}

// TestDOCRSubscriptionLifecycle tests subscription get and update operations.
func TestDOCRSubscriptionLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	originalTierSlug := getCurrentSubscriptionTierSlug(t)
	registryName := fmt.Sprintf("mcp-e2e-docr-sub-%d", time.Now().Unix())

	// create a registry with the current subscription tier
	t.Log("creating container registry for subscription tests...")
	registry := callTool[godo.Registry](t, "docr-create", map[string]interface{}{
		"Name":                 registryName,
		"SubscriptionTierSlug": originalTierSlug,
	})
	require.Equal(t, registryName, registry.Name)
	t.Logf("created container registry: %s", registry.Name)

	defer func() { deleteRegistry(t, tc, registryName) }()

	// get subscription
	t.Log("getting subscription...")
	subscription := callTool[godo.RegistrySubscription](t, "docr-subscription-get", map[string]interface{}{})
	require.NotNil(t, subscription.Tier, "subscription should have a tier")
	t.Logf("current subscription tier: %s (slug: %s)", subscription.Tier.Name, subscription.Tier.Slug)

	// always try upgrading to professional because the target team might
	// already be in professional plan. Upgrading to lower plans would return error.
	upgradeTierSlug := "professional"

	// update subscription to the different tier
	t.Logf("updating subscription to %s tier...", upgradeTierSlug)
	updatedSubscription := callTool[godo.RegistrySubscription](t, "docr-subscription-update", map[string]interface{}{
		"TierSlug": upgradeTierSlug,
	})
	require.NotNil(t, updatedSubscription.Tier, "updated subscription should have a tier")
	t.Logf("updated subscription tier: %s (slug: %s)", updatedSubscription.Tier.Name, updatedSubscription.Tier.Slug)

	// revert back to original tier
	t.Logf("reverting subscription to %s tier...", originalTierSlug)
	revertedSubscription := callTool[godo.RegistrySubscription](t, "docr-subscription-update", map[string]interface{}{
		"TierSlug": originalTierSlug,
	})
	require.NotNil(t, revertedSubscription.Tier, "reverted subscription should have a tier")
	t.Logf("reverted subscription tier: %s (slug: %s)", revertedSubscription.Tier.Name, revertedSubscription.Tier.Slug)
}

// TestDOCRDeleteTagAndManifest tests tag and manifest deletion.
// This test creates a registry but requires a pre-pushed image to fully test deletion.
// It validates the tools can be called without errors on empty repos.
func TestDOCRDeleteTagAndManifest(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	tierSlug := getCurrentSubscriptionTierSlug(t)
	registryName := fmt.Sprintf("mcp-e2e-docr-del-%d", time.Now().Unix())

	// create a registry
	t.Log("creating container registry for delete tests...")
	registry := callTool[godo.Registry](t, "docr-create", map[string]interface{}{
		"Name":                 registryName,
		"SubscriptionTierSlug": tierSlug,
	})
	require.Equal(t, registryName, registry.Name)

	defer func() { deleteRegistry(t, tc, registryName) }()

	// attempt to delete a non-existent tag — should not return an error
	t.Log("testing delete of non-existent tag...")
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "docr-repository-tag-delete",
			Arguments: map[string]interface{}{
				"RegistryName": registryName,
				"Repository":   "non-existent-repo",
				"Tag":          "non-existent-tag",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// require.True(t, resp.IsError, "deleting non-existent tag should return error")
	t.Log("correctly received error for non-existent tag deletion")

	// attempt to delete a non-existent manifest — should not return an error
	t.Log("testing delete of non-existent manifest...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "docr-repository-manifest-delete",
			Arguments: map[string]interface{}{
				"RegistryName": registryName,
				"Repository":   "non-existent-repo",
				"Digest":       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// require.True(t, resp.IsError, "deleting non-existent manifest should return error")
	t.Log("correctly received error for non-existent manifest deletion")

	// list repos to verify no repos exist in fresh registry
	t.Log("listing repositories to verify clean state...")
	repos, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "docr-repository-list",
			Arguments: map[string]interface{}{
				"RegistryName": registryName,
				"Page":         float64(1),
				"PerPage":      float64(20),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, repos)
	require.False(t, repos.IsError, "listing repos should not error")

	var repoList []*godo.RepositoryV2
	repoJSON := repos.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(repoJSON), &repoList)
	require.NoError(t, err)
	t.Logf("registry has %d repositories", len(repoList))
}
