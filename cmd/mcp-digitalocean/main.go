package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	middleware "mcp-digitalocean/internal"
	"mcp-digitalocean/internal/registry"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "1.0.11"
)

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is empty or not present, it returns the fallback value.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logLevelFlag := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", getEnv("SERVICES", ""), "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", getEnv("DIGITALOCEAN_API_TOKEN", ""), "DigitalOcean API token")
	endpointFlag := flag.String("digitalocean-api-endpoint", getEnv("DIGITALOCEAN_API_ENDPOINT", "https://api.digitalocean.com"), "DigitalOcean API endpoint")
	transport := flag.String("transport", getEnv("TRANSPORT", "stdio"), "The transport protocol to use (http or stdio). Default is stdio.")
	bindAddr := flag.String("bind-addr", getEnv("BIND_ADDR", "0.0.0.0:8080"), "Bind address to bind to. Only used for http transport.")
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
		logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
		os.Exit(1)
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	client, err := newGodoClientWithTokenAndEndpoint(context.Background(), token, *endpointFlag)
	if err != nil {
		logger.Error("Failed to create DigitalOcean client: " + err.Error())
		os.Exit(1)
	}

	s := server.NewMCPServer(mcpName, mcpVersion)
	err = registry.Register(logger, s, client, services...)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	err = runServer(s, logger, *bindAddr, transport)
	if err != nil {
		// if context cancelled or sigterm then shutdown gracefully
		if errors.Is(err, context.Canceled) {
			logger.Info("Server shutdown gracefully")
			os.Exit(0)
		} else {
			logger.Error("Failed to serve MCP server: " + err.Error())
			os.Exit(1)
		}
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

func runServer(s *server.MCPServer, logger *slog.Logger, bindAddr string, transport *string) error {
	var err error

	logger.Debug("starting MCP server", "name", mcpName, "version", mcpVersion, "transport", *transport, "bind_addr", bindAddr)
	switch *transport {
	case "http":
		httpServer := server.NewStreamableHTTPServer(
			s,
			server.WithHTTPContextFunc(middleware.AuthFromRequest),
		)

		err = httpServer.Start(bindAddr)
		if err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}

	// stdio is the default transport
	default:
		err = server.ServeStdio(s)
		if err != nil {
			return fmt.Errorf("failed to start STDIO server: %w", err)
		}
	}

	return err
}
