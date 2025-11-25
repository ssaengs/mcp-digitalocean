//go:build integration

package testing

import (
	"slices"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestImageTransfer(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	// 1. Create a user image (snapshot)
	image := CreateTestSnapshotImage(ctx, c, t, "mcp-e2e-img-transfer")
	defer deferCleanupImage(ctx, c, t, float64(image.ID))()

	require.NotEmpty(t, image.Regions)

	// Determine a target region different from current
	currentRegion := image.Regions[0]
	targetRegion := "nyc1"
	if currentRegion == "nyc1" {
		targetRegion = "nyc3"
	}

	t.Logf("Transferring image %d from %s to %s...", image.ID, currentRegion, targetRegion)

	// 2. Trigger Transfer Action
	// Use helper to ensure standard logging of action ID and status
	triggerImageActionAndWait(t, ctx, c, gclient, "image-action-transfer", map[string]any{
		"ID":     float64(image.ID),
		"Region": targetRegion,
	}, image.ID)

	// 3. Verify image has new region
	refreshedImage := callTool[godo.Image](ctx, c, t, "image-get", map[string]any{
		"ID": float64(image.ID),
	})

	found := slices.Contains(refreshedImage.Regions, targetRegion)

	require.True(t, found, "Image does not list target region after transfer")
	t.Logf("Image transfer verified. Regions: %v", refreshedImage.Regions)
}
