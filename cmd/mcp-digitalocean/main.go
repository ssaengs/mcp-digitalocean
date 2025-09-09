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

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", "", "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", "", "DigitalOcean API token")
	transport := flag.String("transport", "stdio", "Transport protocol (http or stdio)")
	bindAddr := flag.String("bind-addr", "0.0.0.0:8080", "Bind address to bind to")

	// optional
	endpointFlag := flag.String("digitalocean-api-endpoint", "", "DigitalOcean API endpoint")
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

	endpoint := *endpointFlag
	if endpoint != "" {
		endpoint = os.Getenv("DIGITALOCEAN_API_ENDPOINT")
		if endpoint == "" {
			endpoint = defaultEndpoint
		}
	}

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
		runHTTPServer(s, logger, *bindAddr, services)
	} else {
		runStdioServer(logger, s, tokenFlag, services)
	}
}

func clientFromApiToken(ctx context.Context) (*godo.Client, error) {
	auth, ok := ctx.Value(authKey{}).(string)
	if !ok || auth == "" {
		return nil, fmt.Errorf("no auth token provided")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, fmt.Errorf("invalid auth token format")
	}
	token := parts[1]

	endpoint := os.Getenv("DIGITALOCEAN_API_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	return newGodoClientWithTokenAndEndpoint(ctx, token, endpoint)
}

func runHTTPServer(s *server.MCPServer, logger *slog.Logger, bindAddr string, services []string) {
	err := registry.Register(logger, s, clientFromApiToken, services...)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	httpServer := server.NewStreamableHTTPServer(s, server.WithHTTPContextFunc(authFromRequest))
	// listen on port 8080
	logger.Debug("Http server start listening: " + bindAddr)
	err = httpServer.Start(bindAddr)
	if err != nil {
		logger.Error("Failed to serve MCP server over HTTP: " + err.Error())
		os.Exit(1)
	}
}

func runStdioServer(logger *slog.Logger, s *server.MCPServer, tokenFlag *string, services []string) {
	err := registry.Register(logger, s, clientFromApiToken, services...)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	// if using stdio, we check for the existence of the env var
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_API_TOKEN")
		if token == "" {
			logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
			os.Exit(1)
		}
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
