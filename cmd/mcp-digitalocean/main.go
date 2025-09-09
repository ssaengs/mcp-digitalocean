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
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_API_TOKEN")
		if token == "" {
			logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
			os.Exit(1)
		}
	}

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

	// The godo-client should be created on a per-request basis, but for the sake of this MCP server.
	client, err := newGodoClientWithTokenAndEndpoint(context.Background(), token, endpoint)
	if err != nil {
		logger.Error("Failed to create DigitalOcean client: " + err.Error())
		os.Exit(1)
	}

	svr := newMcpServer(logger, client, services)
	logger.Debug("starting MCP server", "name", mcpName, "version", mcpVersion)
	if *transport == "http" {
		httpServer := server.NewStreamableHTTPServer(svr, server.WithHTTPContextFunc(authFromRequest))
		// listen on port 8080
		logger.Debug("Http server start listening: " + *bindAddr)
		err = httpServer.Start(*bindAddr)
		if err != nil {
			logger.Error("Failed to serve MCP server over HTTP: " + err.Error())
			os.Exit(1)
		}
	} else {
		logger.Debug("starting stdio server")
		err = server.ServeStdio(svr)
		if err != nil {
			logger.Error("Failed to serve MCP server over stdio: " + err.Error())
			os.Exit(1)
		}
	}
}

func newMcpServer(logger *slog.Logger, client *godo.Client, services []string) *server.MCPServer {
	s := server.NewMCPServer(
		mcpName,
		mcpVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	err := registry.Register(logger, s, client, services...)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	return s
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
