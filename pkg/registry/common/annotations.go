package droplet

import "github.com/mark3labs/mcp-go/mcp"

// hints bundles the four standard MCP tool annotation hints — readOnly,
// destructive, idempotent, openWorld — into a single typed value. Tool
// registrations should reference one of the profile vars below
// (hintsRead, hintsAction, hintsToggle, hintsDelete, hintsReplace) rather
// than constructing a hints literal at the call site; the profile names
// document the intent at the registration point and make it impossible to
// swap two bool arguments.
type hints struct {
	readOnly, destructive, idempotent, openWorld bool
}

// apply sets all four MCP tool hint annotations on t, overriding the mcp-go
// library defaults (which assume the worst case: not read-only, destructive,
// not idempotent).
func (h hints) apply(t *mcp.Tool) {
	mcp.WithReadOnlyHintAnnotation(h.readOnly)(t)
	mcp.WithDestructiveHintAnnotation(h.destructive)(t)
	mcp.WithIdempotentHintAnnotation(h.idempotent)(t)
	mcp.WithOpenWorldHintAnnotation(h.openWorld)(t)
}

// withHints returns a ToolOption that applies the given hint profile to a
// tool registration.
func withHints(h hints) mcp.ToolOption { return h.apply }

// Hint profiles. Every tool in this package falls into one of five
// categories; openWorld is true on every profile because every tool
// interacts with the DigitalOcean API.
//
// Delete is destructive AND idempotent: repeating a delete on an
// already-removed resource returns not-found with no additional side
// effect. Replace (rebuild, restore) is destructive but non-idempotent
// because each call kicks off a fresh action that replaces droplet state.
var (
	// hintsRead — read-only list/get/inspect tools. Idempotent because
	// observing state again returns the same data.
	hintsRead = hints{readOnly: true, idempotent: true, openWorld: true}

	// hintsAction — non-destructive mutation that does not converge on a
	// target state; each call kicks off a fresh action (reboot, snapshot,
	// password reset, droplet create).
	hintsAction = hints{openWorld: true}

	// hintsToggle — mutation that converges on a target state. Repeated
	// calls have no additional effect (power-on twice = still on; rename
	// to the same name = no-op).
	hintsToggle = hints{idempotent: true, openWorld: true}

	// hintsDelete — destructive AND idempotent (see package note above).
	hintsDelete = hints{destructive: true, idempotent: true, openWorld: true}

	// hintsReplace — destructive AND non-idempotent (see package note above).
	hintsReplace = hints{destructive: true, openWorld: true}
)
