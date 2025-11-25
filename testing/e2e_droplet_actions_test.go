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

func TestDropletReboot(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-reboot")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	triggerActionAndWait(t, ctx, c, gclient, "reboot-droplet", map[string]interface{}{"ID": d.ID}, d.ID)
}

func TestDropletPowerCycle(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-powercycle")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	triggerActionAndWait(t, ctx, c, gclient, "power-cycle-droplet", map[string]interface{}{"ID": d.ID}, d.ID)
}

func TestDropletSnapshotAction(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-snap-action")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	// Trigger Snapshot
	snapName := fmt.Sprintf("e2e-snap-%d", time.Now().Unix())
	triggerActionAndWait(t, ctx, c, gclient, "snapshot-droplet", map[string]interface{}{
		"ID":   d.ID,
		"Name": snapName,
	}, d.ID)

	// Verify Snapshot Existence via Direct API
	refreshed, err := testhelpers.WaitForDroplet(ctx, gclient, d.ID, func(d *godo.Droplet) bool {
		return d != nil && len(d.SnapshotIDs) > 0
	}, 3*time.Second, 2*time.Minute)

	require.NoError(t, err, "Failed to verify snapshot creation")
	require.NotEmpty(t, refreshed.SnapshotIDs)

	// Cleanup the created snapshot image
	imageID := float64(refreshed.SnapshotIDs[0])
	t.Logf("Snapshot verified. Cleaning up Image ID: %.0f", imageID)
	defer deferCleanupImage(ctx, c, t, imageID)()
}

func TestDropletRename(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-rename")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	// Trigger Rename
	newName := fmt.Sprintf("%s-renamed", d.Name)
	triggerActionAndWait(t, ctx, c, gclient, "rename-droplet", map[string]interface{}{
		"ID":   d.ID,
		"Name": newName,
	}, d.ID)

	// Verify Name Change
	refreshed, err := testhelpers.WaitForDroplet(ctx, gclient, d.ID, func(droplet *godo.Droplet) bool {
		return droplet != nil && droplet.Name == newName
	}, 2*time.Second, 30*time.Second)

	require.NoError(t, err, "Failed to verify rename")
	require.Equal(t, newName, refreshed.Name)

	t.Logf("Droplet successfully renamed to: %s", refreshed.Name)
}

func TestDropletEnableIPv6(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-ipv6")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	triggerActionAndWait(t, ctx, c, gclient, "enable-ipv6-droplet", map[string]interface{}{"ID": d.ID}, d.ID)

	// Verify IPv6 Address Assignment
	refreshed, err := testhelpers.WaitForDroplet(ctx, gclient, d.ID, func(droplet *godo.Droplet) bool {
		return droplet != nil && len(droplet.Networks.V6) > 0
	}, 3*time.Second, 1*time.Minute)

	require.NoError(t, err, "IPv6 address was not assigned within timeout")
	t.Logf("IPv6 Assigned: %s", refreshed.Networks.V6[0].IPAddress)
}

func TestDropletEnableBackups(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-backups")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	triggerActionAndWait(t, ctx, c, gclient, "enable-backups-droplet", map[string]interface{}{"ID": d.ID}, d.ID)

	// Verify Backups Enabled
	refreshed, err := testhelpers.WaitForDroplet(ctx, gclient, d.ID, func(droplet *godo.Droplet) bool {
		if droplet == nil {
			return false
		}
		// Check standard fields
		if len(droplet.BackupIDs) > 0 || droplet.NextBackupWindow != nil {
			return true
		}
		// Fallback: Check explicit policy
		policy := callTool[godo.DropletBackupPolicy](ctx, c, t, "droplet-backup-policy", map[string]interface{}{"ID": droplet.ID})
		return policy.BackupEnabled
	}, 3*time.Second, 1*time.Minute)

	require.NoError(t, err, "Backups were not enabled within timeout")
	t.Logf("Backups Verified for Droplet %d", refreshed.ID)
}

func TestDropletActionTool(t *testing.T) {
	t.Parallel()

	ctx, c, gclient, teardown := setupTest(t)
	defer teardown()

	d := CreateTestDroplet(ctx, c, t, "mcp-e2e-action-tool")
	defer deferCleanupDroplet(ctx, c, t, d.ID)()

	cycleAction := callTool[godo.Action](ctx, c, t, "power-cycle-droplet", map[string]interface{}{"ID": d.ID})
	require.NotZero(t, cycleAction.ID)

	fetchedAction := callTool[godo.Action](ctx, c, t, "droplet-action", map[string]interface{}{
		"DropletID": float64(d.ID),
		"ActionID":  float64(cycleAction.ID),
	})

	t.Logf("Retrieved Action via Tool: ID=%d, Type=%s, Status=%s", fetchedAction.ID, fetchedAction.Type, fetchedAction.Status)

	require.Equal(t, cycleAction.ID, fetchedAction.ID)
	require.Equal(t, "power_cycle", fetchedAction.Type)

	final, err := testhelpers.WaitForAction(ctx, gclient, d.ID, cycleAction.ID, 2*time.Second, 2*time.Minute)
	require.NoError(t, err)

	LogActionStatus(t, "ActionToolTest", *final)
}
