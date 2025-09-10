package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	registry "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "1.0.10"

	defaultEndpoint = "https://api.digitalocean.com"
)

type authKey struct{}

// authFromRequest extracts the auth token from the request headers.
func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	return withAuthKey(ctx, r.Header.Get("Authorization"))
}

// withAuthKey adds an auth key to the context.
func withAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logLevelFlag := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", getEnv("SERVICES", ""), "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", getEnv("DIGITALOCEAN_API_TOKEN", ""), "DigitalOcean API token. If not provided, will use DIGITALOCEAN_API_TOKEN environment variable. This is only used for stdio transport.")
	transport := flag.String("transport", getEnv("TRANSPORT", "stdio"), "Transport protocol (http or stdio)")
	bindAddr := flag.String("bind-addr", getEnv("BIND_ADDR", "0.0.0.0:8080"), "Bind address to bind to. Only used for http transport.")
	endpointFlag := flag.String("digitalocean-api-endpoint", getEnv("DIGITALOCEAN_API_ENDPOINT", defaultEndpoint), "DigitalOcean API endpoint")
	flag.Parse()

	var level slog.Level
	switch strings.ToLower(*logLevelFlag) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	s := server.NewMCPServer(
		mcpName,
		mcpVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	logger.Debug("starting MCP server", "name", mcpName, "version", mcpVersion)
	if *transport == "http" {
		runHTTPServer(s, logger, *bindAddr, endpointFlag, services)
	} else {
		runStdioServer(s, logger, tokenFlag, endpointFlag, services)
	}
}

// clientFromContext creates a godo client from authentication info in the context.
func clientFromContext(ctx context.Context, endpoint string) *godo.Client {
	auth, ok := ctx.Value(authKey{}).(string)
	if !ok || strings.TrimSpace(auth) == "" {
		return nil
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return nil
	}
	client, err := newGodoClientWithTokenAndEndpoint(ctx, token, endpoint)
	if err != nil {
		return nil
	}

	return client
}

// clientFromApiToken creates a godo client from a static API token provided via flag or environment variable.
func clientFromApiToken(ctx context.Context, token, endpoint string) *godo.Client {
	client, err := newGodoClientWithTokenAndEndpoint(ctx, token, endpoint)
	if err != nil {
		return nil
	}
	return client
}

func runHTTPServer(s *server.MCPServer, logger *slog.Logger, bindAddr string, endpointFlag *string, services []string) {
	err := registry.Register(
		logger,
		s,
		func(ctx context.Context) *godo.Client {
			return clientFromContext(ctx, *endpointFlag)
		},
		services...,
	)

	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	httpServer := server.NewStreamableHTTPServer(s, server.WithHTTPContextFunc(authFromRequest))
	logger.Debug("Http server start listening: " + bindAddr)
	err = httpServer.Start(bindAddr)
	if err != nil {
		logger.Error("Failed to serve MCP server over HTTP: " + err.Error())
		os.Exit(1)
	}
}

func runStdioServer(s *server.MCPServer, logger *slog.Logger, tokenFlag *string, endpointFlag *string, services []string) {
	err := registry.Register(
		logger,
		s,
		func(ctx context.Context) *godo.Client {
			return clientFromApiToken(ctx, *tokenFlag, *endpointFlag)
		},
		services...,
	)

	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	logger.Debug("starting stdio server")
	err = server.ServeStdio(s)
	if err != nil {
		logger.Error("Failed to serve MCP server over stdio: " + err.Error())
		os.Exit(1)
	}
}

// newGodoClientWithTokenAndEndpoint initializes a new godo client with a custom user agent and endpoint.
func newGodoClientWithTokenAndEndpoint(ctx context.Context, token string, endpoint string) (*godo.Client, error) {
	cleanToken := strings.Trim(strings.TrimSpace(token), "'")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cleanToken})
	oauthClient := oauth2.NewClient(ctx, ts)

	retry := godo.RetryConfig{
		RetryMax:     4,
		RetryWaitMin: godo.PtrTo(float64(1)),
		RetryWaitMax: godo.PtrTo(float64(30)),
	}

	return godo.New(oauthClient,
		godo.WithRetryAndBackoffs(retry),
		godo.SetBaseURL(endpoint),
		godo.SetUserAgent(fmt.Sprintf("%s/%s", mcpName, mcpVersion)))
}
