package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/internal/account"
	"mcp-digitalocean/internal/apps"
	"mcp-digitalocean/internal/common"
	"mcp-digitalocean/internal/dbaas"
	"mcp-digitalocean/internal/doks"
	"mcp-digitalocean/internal/droplet"
	"mcp-digitalocean/internal/insights"
	"mcp-digitalocean/internal/marketplace"
	"mcp-digitalocean/internal/networking"
	"mcp-digitalocean/internal/spaces"
)

// supportedServices is a set of services that we support in this MCP server.
var supportedServices = map[string]struct{}{
	"apps":        {},
	"networking":  {},
	"droplets":    {},
	"accounts":    {},
	"spaces":      {},
	"databases":   {},
	"marketplace": {},
	"insights":    {},
	"doks":        {},
}

// TODO, this function should return client and an error.
type getClientFn func(ctx context.Context) *godo.Client

// registerAppTools registers the app platform tools with the MCP server.
func registerAppTools(s *server.MCPServer, fn getClientFn) error {
	appTools, err := apps.NewAppPlatformTool(fn)
	if err != nil {
		return fmt.Errorf("failed to create apps tool: %w", err)
	}

	s.AddTools(appTools.Tools()...)

	return nil
}

// registerCommonTools registers the common tools with the MCP server.
func registerCommonTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(common.NewRegionTools(fn).Tools()...)

	return nil
}

// registerDropletTools registers the droplet tools with the MCP server.
func registerDropletTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(droplet.NewDropletTool(fn).Tools()...)
	s.AddTools(droplet.NewDropletActionsTool(fn).Tools()...)
	s.AddTools(droplet.NewImagesTool(fn).Tools()...)
	s.AddTools(droplet.NewSizesTool(fn).Tools()...)
	return nil
}

// registerNetworkingTools registers the networking tools with the MCP server.
func registerNetworkingTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(networking.NewCertificateTool(fn).Tools()...)
	s.AddTools(networking.NewDomainsTool(fn).Tools()...)
	s.AddTools(networking.NewFirewallTool(fn).Tools()...)
	s.AddTools(networking.NewReservedIPTool(fn).Tools()...)
	s.AddTools(networking.NewVPCTool(fn).Tools()...)
	s.AddTools(networking.NewVPCPeeringTool(fn).Tools()...)
	return nil
}

// registerAccountTools registers the account tools with the MCP server.
func registerAccountTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(account.NewAccountTools(fn).Tools()...)
	s.AddTools(account.NewActionTools(fn).Tools()...)
	s.AddTools(account.NewBalanceTools(fn).Tools()...)
	s.AddTools(account.NewBillingTools(fn).Tools()...)
	s.AddTools(account.NewInvoiceTools(fn).Tools()...)
	s.AddTools(account.NewKeysTool(fn).Tools()...)

	return nil
}

// registerSpacesTools registers the spaces tools and resources with the MCP server.
func registerSpacesTools(s *server.MCPServer, fn getClientFn) error {
	// Register the tools for spaces keys
	s.AddTools(spaces.NewSpacesKeysTool(fn).Tools()...)
	s.AddTools(spaces.NewCDNTool(fn).Tools()...)

	return nil
}

// registerMarketplaceTools registers the marketplace tools with the MCP server.
func registerMarketplaceTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(marketplace.NewOneClickTool(fn).Tools()...)

	return nil
}

func registerInsightsTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(insights.NewUptimeTool(fn).Tools()...)
	s.AddTools(insights.NewUptimeCheckAlertTool(fn).Tools()...)
	s.AddTools(insights.NewAlertPolicyTool(fn).Tools()...)
	return nil
}

func registerDOKSTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(doks.NewDoksTool(fn).Tools()...)

	return nil
}

func registerDatabasesTools(s *server.MCPServer, fn getClientFn) error {
	s.AddTools(dbaas.NewClusterTool(fn).Tools()...)
	s.AddTools(dbaas.NewFirewallTool(fn).Tools()...)
	s.AddTools(dbaas.NewKafkaTool(fn).Tools()...)
	s.AddTools(dbaas.NewMongoTool(fn).Tools()...)
	s.AddTools(dbaas.NewMysqlTool(fn).Tools()...)
	s.AddTools(dbaas.NewOpenSearchTool(fn).Tools()...)
	s.AddTools(dbaas.NewPostgreSQLTool(fn).Tools()...)
	s.AddTools(dbaas.NewRedisTool(fn).Tools()...)
	s.AddTools(dbaas.NewUserTool(fn).Tools()...)

	return nil
}

// Register registers the set of tools for the specified services with the MCP server.
// We either register a subset of tools of the services are specified, or we register all tools if no services are specified.
func Register(logger *slog.Logger, s *server.MCPServer, fn getClientFn, servicesToActivate ...string) error {
	if len(servicesToActivate) == 0 {
		logger.Warn("no services specified, loading all supported services")
		for k := range supportedServices {
			servicesToActivate = append(servicesToActivate, k)
		}
	}
	for _, svc := range servicesToActivate {
		logger.Debug(fmt.Sprintf("Registering tool and resources for service: %s", svc))
		switch svc {
		case "apps":
			if err := registerAppTools(s, fn); err != nil {
				return fmt.Errorf("failed to register app tools: %w", err)
			}
		case "networking":
			if err := registerNetworkingTools(s, fn); err != nil {
				return fmt.Errorf("failed to register networking tools: %w", err)
			}
		case "droplets":
			if err := registerDropletTools(s, fn); err != nil {
				return fmt.Errorf("failed to register droplets tool: %w", err)
			}
		case "accounts":
			if err := registerAccountTools(s, fn); err != nil {
				return fmt.Errorf("failed to register account tools: %w", err)
			}
		case "spaces":
			if err := registerSpacesTools(s, fn); err != nil {
				return fmt.Errorf("failed to register spaces tools: %w", err)
			}
		case "databases":
			if err := registerDatabasesTools(s, fn); err != nil {
				return fmt.Errorf("failed to register databases tools: %w", err)
			}
		case "marketplace":
			if err := registerMarketplaceTools(s, fn); err != nil {
				return fmt.Errorf("failed to register marketplace tools: %w", err)
			}
		case "insights":
			if err := registerInsightsTools(s, fn); err != nil {
				return fmt.Errorf("failed to register insights tools: %w", err)
			}
		case "doks":
			if err := registerDOKSTools(s, fn); err != nil {
				return fmt.Errorf("failed to register DOKS tools: %w", err)
			}
		default:
			return fmt.Errorf("unsupported service: %s, supported service are: %v", svc, setToString(supportedServices))
		}
	}

	// Common tools are always registered because they provide common functionality for all services such as region resources
	if err := registerCommonTools(s, fn); err != nil {
		return fmt.Errorf("failed to register common tools: %w", err)
	}

	return nil
}

func setToString(set map[string]struct{}) string {
	var result []string
	for key := range set {
		result = append(result, key)
	}

	return strings.Join(result, ",")
}
