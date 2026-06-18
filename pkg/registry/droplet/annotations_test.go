package droplet

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
)

// expectedAnnotations is the per-tool annotation contract enforced by
// TestToolAnnotations. Adding a tool to this package WITHOUT a row here, or
// changing an annotation in code without updating the row, fails the test.
//
// Order: readOnly, destructive, idempotent, openWorld.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
}{
	// droplet_tools.go
	"droplet-create":             {false, false, false, true},
	"droplet-delete":             {false, true, true, true},
	"droplet-enable-private-net": {false, false, true, true},
	"droplet-kernels":            {true, false, true, true},
	"droplet-get":                {true, false, true, true},
	"droplet-backup-policy":      {true, false, true, true},
	"droplet-action":             {true, false, true, true},
	"droplet-list":               {true, false, true, true},

	// droplet_actions_tools.go
	"reboot-droplet":                  {false, false, false, true},
	"reset-droplet-password":          {false, false, false, true},
	"rebuild-droplet-by-slug":         {false, true, false, true},
	"power-cycle-droplets-tag":        {false, false, false, true},
	"power-on-droplets-tag":           {false, false, true, true},
	"power-off-droplets-tag":          {false, false, true, true},
	"shutdown-droplets-tag":           {false, false, true, true},
	"enable-backups-droplets-tag":     {false, false, true, true},
	"disable-backups-droplets-tag":    {false, false, true, true},
	"snapshot-droplets-tag":           {false, false, false, true},
	"enable-ipv6-droplets-tag":        {false, false, true, true},
	"enable-private-net-droplets-tag": {false, false, true, true},
	"power-cycle-droplet":             {false, false, false, true},
	"power-on-droplet":                {false, false, true, true},
	"power-off-droplet":               {false, false, true, true},
	"shutdown-droplet":                {false, false, true, true},
	"restore-droplet":                 {false, true, false, true},
	"resize-droplet":                  {false, false, false, true},
	"rebuild-droplet":                 {false, true, false, true},
	"rename-droplet":                  {false, false, true, true},
	"change-kernel-droplet":           {false, false, true, true},
	"enable-ipv6-droplet":             {false, false, true, true},
	"enable-backups-droplet":          {false, false, true, true},
	"disable-backups-droplet":         {false, false, true, true},
	"snapshot-droplet":                {false, false, false, true},

	// image_actions_tools.go
	"image-action-transfer": {false, false, true, true},
	"image-action-convert":  {false, false, true, true},
	"image-action-get":      {true, false, true, true},

	// images_tools.go
	"image-list":   {true, false, true, true},
	"image-get":    {true, false, true, true},
	"image-create": {false, false, false, true},
	"image-update": {false, false, true, true},
	"image-delete": {false, true, true, true},

	// sizes_tools.go
	"size-list": {true, false, true, true},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, NewDropletTool(clientFn).Tools()...)
	all = append(all, NewDropletActionsTool(clientFn).Tools()...)
	all = append(all, NewImageActionsTool(clientFn).Tools()...)
	all = append(all, NewImageTool(clientFn).Tools()...)
	all = append(all, NewSizesTool(clientFn).Tools()...)

	if len(all) != len(expectedAnnotations) {
		t.Fatalf("tool count mismatch: registered=%d, expected=%d (add new tools to expectedAnnotations)", len(all), len(expectedAnnotations))
	}

	seen := make(map[string]bool, len(all))
	for _, st := range all {
		name := st.Tool.Name
		if seen[name] {
			t.Errorf("duplicate tool registration: %q", name)
			continue
		}
		seen[name] = true

		want, ok := expectedAnnotations[name]
		if !ok {
			t.Errorf("tool %q has no expected annotation row — add one to expectedAnnotations", name)
			continue
		}

		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || a.DestructiveHint == nil || a.IdempotentHint == nil || a.OpenWorldHint == nil {
			t.Errorf("tool %q is missing one or more annotation hints (got %+v)", name, a)
			continue
		}
		if got := *a.ReadOnlyHint; got != want.readOnly {
			t.Errorf("tool %q readOnlyHint = %v, want %v", name, got, want.readOnly)
		}
		if got := *a.DestructiveHint; got != want.destructive {
			t.Errorf("tool %q destructiveHint = %v, want %v", name, got, want.destructive)
		}
		if got := *a.IdempotentHint; got != want.idempotent {
			t.Errorf("tool %q idempotentHint = %v, want %v", name, got, want.idempotent)
		}
		if got := *a.OpenWorldHint; got != want.openWorld {
			t.Errorf("tool %q openWorldHint = %v, want %v", name, got, want.openWorld)
		}
	}
}
