package droplet

import "github.com/mark3labs/mcp-go/mcp"

// withHints returns a ToolOption that sets all four standard MCP tool hint
// annotations explicitly, overriding the mcp-go library defaults (which assume
// the worst case: not read-only, destructive, not idempotent). Every tool in
// this package interacts with the DigitalOcean API, so openWorld is always
// true; we still pass it for clarity and to lock the value down.
func withHints(readOnly, destructive, idempotent, openWorld bool) mcp.ToolOption {
	return func(t *mcp.Tool) {
		mcp.WithReadOnlyHintAnnotation(readOnly)(t)
		mcp.WithDestructiveHintAnnotation(destructive)(t)
		mcp.WithIdempotentHintAnnotation(idempotent)(t)
		mcp.WithOpenWorldHintAnnotation(openWorld)(t)
	}
}
