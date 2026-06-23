package common

import "github.com/mark3labs/mcp-go/mcp"

// hints bundles the four standard MCP tool annotation hints — readOnly,
// destructive, idempotent, openWorld — into a single typed value. Tool
// registrations should reference one of the profile vars below
// (HintsRead, HintsAction, HintsToggle, HintsDelete, HintsReplace) rather
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

// WithHints returns a ToolOption that applies the given hint profile to a
// tool registration.
func WithHints(h hints) mcp.ToolOption { return h.apply }

// Hint profiles. Every tool in this package falls into one of five
// categories; openWorld is true on every profile because every tool
// interacts with the DigitalOcean API.
//
// Delete is destructive AND idempotent: repeating a delete on an
// already-removed resource returns not-found with no additional side
// effect. Replace (rebuild, restore) is destructive but non-idempotent
// because each call kicks off a fresh action that replaces droplet state.
var (
	// HintsRead — read-only list/get/inspect tools. Idempotent because
	// observing state again returns the same data.
	HintsRead = hints{readOnly: true, idempotent: true, openWorld: false}

	// HintsAction — non-destructive mutation that does not converge on a
	// target state; each call kicks off a fresh action (reboot, snapshot,
	// password reset, droplet create).
	HintsAction = hints{openWorld: false}

	// HintsToggle — mutation that converges on a target state. Repeated
	// calls have no additional effect (power-on twice = still on; rename
	// to the same name = no-op).
	HintsToggle = hints{idempotent: true, openWorld: false}

	// HintsDelete — destructive AND idempotent (see package note above).
	HintsDelete = hints{destructive: true, idempotent: true, openWorld: false}

	// HintsReplace — destructive AND non-idempotent (see package note above).
	HintsReplace = hints{destructive: true, openWorld: false}
)
