//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"mcp-digitalocean/internal/testhelpers"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestImageList(t *testing.T) {
	t.Parallel()

	ctx, c, _, teardown := setupTest(t)
	defer teardown()

	perPage := 20
	t.Logf("[List] Requesting distribution images (PerPage: %d)...", perPage)

	// List distribution images
	images := callTool[[]map[string]any](ctx, c, t, "image-list", map[string]any{
		"Type":    "distribution",
		"PerPage": perPage,
	})

	require.NotEmpty(t, images)
	// DigitalOcean API returns "base" for distribution images now
	require.Equal(t, "base", images[0]["type"])

	t.Logf("[List] Successfully listed %d distribution images:", len(images))
	for _, img := range images {
		name, _ := img["name"].(string)
		slug, _ := img["slug"].(string)
		id, _ := img["id"].(float64)
		t.Logf(" - %s (ID: %.0f, Slug: %s)", name, id, slug)
	}
}

func TestImageGet(t *testing.T) {
	t.Parallel()

	ctx, c, _, teardown := setupTest(t)
	defer teardown()

	// Get a distribution image ID first
	id, slug := getTestImage(ctx, c, t)

	t.Logf("[Get] Requesting details for Image ID: %.0f (Slug: %s)...", id, slug)

	image := callTool[godo.Image](ctx, c, t, "image-get", map[string]any{
		"ID": id,
	})

	require.Equal(t, int(id), image.ID)
	require.NotEmpty(t, image.Slug)

	t.Logf("[Get] Successfully retrieved image:")
	t.Logf("      Name: %s", image.Name)
	t.Logf("      ID: %d", image.ID)
	t.Logf("      Slug: %s", image.Slug)
	t.Logf("      Distribution: %s", image.Distribution)
	t.Logf("      Type: %s", image.Type)
	t.Logf("      Regions: %v", image.Regions)
}

func TestImageLifecycle(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	// 1. Create a user image (via snapshot of a droplet)
	image := CreateTestSnapshotImage(ctx, c, t, "mcp-e2e-image-lifecycle")
	defer deferCleanupImage(ctx, c, t, float64(image.ID))()

	// 2. Update Image (Rename)
	newName := fmt.Sprintf("%s-renamed", image.Name)
	t.Logf("[Update] Renaming image %d to %s...", image.ID, newName)

	updatedImage := callTool[godo.Image](ctx, c, t, "image-update", map[string]any{
		"ID":   float64(image.ID),
		"Name": newName,
	})

	require.Equal(t, newName, updatedImage.Name)
	t.Logf("[Update] Success. New Name: %s", updatedImage.Name)

	// 3. Delete Image
	// Using "image" resource type maps to "image-delete" tool via e2e_helpers logic
	DeleteResource(ctx, c, t, "image", float64(image.ID))

	// 4. Verify Deletion
	t.Logf("[Verify] Waiting for image %d deletion...", image.ID)
	err := testhelpers.WaitForImageDeleted(ctx, gclient, image.ID, 2*time.Second, 1*time.Minute)
	require.NoError(t, err, "Image was not deleted")
	t.Logf("[Verify] Confirmed image %d deletion", image.ID)
}
