package common

import (
	"context"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"mcp-digitalocean/pkg/registry/droplet"
)

// expectedAnnotations is the per-tool annotation contract enforced by
// TestToolAnnotations. Adding a tool to this package WITHOUT a row here, or
// changing an annotation in code without updating the row, fails the test.
//
// Order: readOnly, destructive, idempotent, openWorld. See annotations.go
// for the rationale behind the five profile categories these rows map to.
var expectedAnnotations = map[string]struct {
	readOnly, destructive, idempotent, openWorld bool
}{
	// droplet_tools.go
	"droplet-create":             {false, false, false, false},
	"droplet-delete":             {false, true, true, false},
	"droplet-enable-private-net": {false, false, true, false},
	"droplet-kernels":            {true, false, true, false},
	"droplet-get":                {true, false, true, false},
	"droplet-backup-policy":      {true, false, true, false},
	"droplet-action":             {true, false, true, false},
	"droplet-list":               {true, false, true, false},

	// droplet_actions_tools.go
	"reboot-droplet":                  {false, false, false, false},
	"reset-droplet-password":          {false, false, false, false},
	"rebuild-droplet-by-slug":         {false, true, false, false},
	"power-cycle-droplets-tag":        {false, false, false, false},
	"power-on-droplets-tag":           {false, false, true, false},
	"power-off-droplets-tag":          {false, false, true, false},
	"shutdown-droplets-tag":           {false, false, true, false},
	"enable-backups-droplets-tag":     {false, false, true, false},
	"disable-backups-droplets-tag":    {false, false, true, false},
	"snapshot-droplets-tag":           {false, false, false, false},
	"enable-ipv6-droplets-tag":        {false, false, true, false},
	"enable-private-net-droplets-tag": {false, false, true, false},
	"power-cycle-droplet":             {false, false, false, false},
	"power-on-droplet":                {false, false, true, false},
	"power-off-droplet":               {false, false, true, false},
	"shutdown-droplet":                {false, false, true, false},
	"restore-droplet":                 {false, true, false, false},
	"resize-droplet":                  {false, false, true, false},
	"rebuild-droplet":                 {false, true, false, false},
	"rename-droplet":                  {false, false, true, false},
	"change-kernel-droplet":           {false, false, true, false},
	"enable-ipv6-droplet":             {false, false, true, false},
	"enable-backups-droplet":          {false, false, true, false},
	"disable-backups-droplet":         {false, false, true, false},
	"snapshot-droplet":                {false, false, false, false},

	// image_actions_tools.go
	"image-action-transfer": {false, false, true, false},
	"image-action-convert":  {false, false, true, false},
	"image-action-get":      {true, false, true, false},

	// images_tools.go
	"image-list":   {true, false, true, false},
	"image-get":    {true, false, true, false},
	"image-create": {false, false, false, false},
	"image-update": {false, false, true, false},
	"image-delete": {false, true, true, false},

	// sizes_tools.go
	"size-list": {true, false, true, false},
}

func TestToolAnnotations(t *testing.T) {
	clientFn := func(context.Context) (*godo.Client, error) {
		return godo.NewFromToken("test-token"), nil
	}

	var all []server.ServerTool
	all = append(all, droplet.NewDropletTool(clientFn).Tools()...)
	all = append(all, droplet.NewDropletActionsTool(clientFn).Tools()...)
	all = append(all, droplet.NewImageActionsTool(clientFn).Tools()...)
	all = append(all, droplet.NewImageTool(clientFn).Tools()...)
	all = append(all, droplet.NewSizesTool(clientFn).Tools()...)

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
