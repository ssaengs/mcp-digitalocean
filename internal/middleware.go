package middleware

import (
	"context"
	"net/http"
)

type AuthKey struct{}

// AuthFromRequest extracts the auth token from the request headers.
func AuthFromRequest(ctx context.Context, r *http.Request) context.Context {
	return WithAuthKey(ctx, r.Header.Get("Authorization"))
}

// WithAuthKey adds an auth key to the context.
func WithAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, AuthKey{}, auth)
}
