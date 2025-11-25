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

func TestDropletLifecycle(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-test")

	type dropletShort struct {
		ID int `json:"id"`
	}
	droplets := callTool[[]dropletShort](ctx, c, t, "droplet-list", map[string]interface{}{"Page": 1, "PerPage": 50})

	found := false
	for _, d := range droplets {
		if d.ID == droplet.ID {
			found = true
			break
		}
	}
	require.True(t, found, "Created droplet not found in list")

	DeleteResource(ctx, c, t, "droplet", droplet.ID)

	err := testhelpers.WaitForDropletDeleted(ctx, gclient, droplet.ID, 3*time.Second, 2*time.Minute)
	if err != nil {
		t.Logf("Warning: direct WaitForDropletDeleted failed: %v", err)
	} else {
		t.Logf("Confirmed droplet deletion via direct API")
	}
}

func TestDropletSnapshot(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-snapshot")
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()

	snapshotName := fmt.Sprintf("snapshot-%d", time.Now().Unix())
	action := callTool[godo.Action](ctx, c, t, "snapshot-droplet", map[string]interface{}{
		"ID":   droplet.ID,
		"Name": snapshotName,
	})

	// Log 1: Initial
	LogActionStatus(t, "Snapshot", action)

	// Log 2: Final (returned by WaitForActionComplete)
	completed := WaitForActionComplete(ctx, c, t, droplet.ID, action.ID, 2*time.Minute)
	LogActionStatus(t, "Snapshot", completed)

	d, err := testhelpers.WaitForDroplet(ctx, gclient, droplet.ID, func(d *godo.Droplet) bool {
		return d != nil && len(d.SnapshotIDs) > 0
	}, 3*time.Second, 2*time.Minute)

	require.NoError(t, err)
	require.NotEmpty(t, d.SnapshotIDs)

	t.Logf("Snapshot verified. Image ID: %d", d.SnapshotIDs[0])
	defer deferCleanupImage(ctx, c, t, float64(d.SnapshotIDs[0]))()
}

func TestDropletRebuildBySlug(t *testing.T) {
	t.Parallel()

	ctx, c, _, teardown := setupTest(t)
	defer teardown()

	type imageShort struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}
	images := callTool[[]imageShort](ctx, c, t, "image-list", map[string]interface{}{"Page": 1, "PerPage": 20, "Type": "distribution"})

	var imageSlug string
	for _, img := range images {
		if img.Slug == "ubuntu-22-04-x64" {
			imageSlug = img.Slug
			break
		}
	}
	if imageSlug == "" && len(images) > 0 {
		imageSlug = images[0].Slug
	}
	if imageSlug == "" {
		t.Skip("No suitable image slug found")
	}

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-rebuild")
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()

	action := callTool[godo.Action](ctx, c, t, "rebuild-droplet-by-slug", map[string]interface{}{
		"ID":        droplet.ID,
		"ImageSlug": imageSlug,
	})

	// Log 1: Initial
	LogActionStatus(t, "Rebuild", action)

	// Log 2: Final
	completed := WaitForActionComplete(ctx, c, t, droplet.ID, action.ID, 5*time.Minute)
	LogActionStatus(t, "Rebuild", completed)
}

func TestDropletRestore(t *testing.T) {
	t.Parallel()

	ctx, c, _, teardown := setupTest(t)
	defer teardown()

	droplet := CreateTestDroplet(ctx, c, t, "mcp-e2e-restore")
	defer deferCleanupDroplet(ctx, c, t, droplet.ID)()

	// Create Snapshot
	snapName := fmt.Sprintf("restore-snap-%d", time.Now().Unix())
	snapAction := callTool[godo.Action](ctx, c, t, "snapshot-droplet", map[string]interface{}{
		"ID":   droplet.ID,
		"Name": snapName,
	})
	WaitForActionComplete(ctx, c, t, droplet.ID, snapAction.ID, 2*time.Minute)

	refreshed := callTool[godo.Droplet](ctx, c, t, "droplet-get", map[string]interface{}{"ID": droplet.ID})
	require.NotEmpty(t, refreshed.SnapshotIDs, "Droplet should have a snapshot")

	imageID := float64(refreshed.SnapshotIDs[0])
	defer deferCleanupImage(ctx, c, t, imageID)()
	t.Logf("Restoring from Image ID: %.0f", imageID)

	// Restore
	restoreAction := callTool[godo.Action](ctx, c, t, "restore-droplet", map[string]interface{}{
		"ID":      droplet.ID,
		"ImageID": imageID,
	})

	// Log 1: Initial
	LogActionStatus(t, "Restore", restoreAction)

	// Log 2: Final
	completed := WaitForActionComplete(ctx, c, t, droplet.ID, restoreAction.ID, 2*time.Minute)
	LogActionStatus(t, "Restore", completed)
}
