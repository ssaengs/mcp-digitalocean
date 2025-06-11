package main

import (
	"log/slog"
	"os"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"

	registry "mcp-digitalocean/internal"
)

const (
	mcpName    = "mcp-digitalocean"
	mcpVersion = "0.1.0"
)

func main() {
	// Read OAUTH token from environment
	token := os.Getenv("DIGITALOCEAN_API_TOKEN")
	if token == "" {
		slog.Error("DIGITALOCEAN_API_TOKEN environment variable is not set")
		os.Exit(1)
	}

	client := godo.NewFromToken(token)
	s := server.NewMCPServer(mcpName, mcpVersion)

	// Register the tools and resources
	registry.RegisterTools(s, client)
	registry.RegisterResources(s, client)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

}
