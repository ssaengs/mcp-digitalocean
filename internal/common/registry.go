package common

import (
	"fmt"
	"log/slog"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"

	"mcp-digitalocean/internal/networking/resources"
	"mcp-digitalocean/internal/networking/tools"
)

// supportedServices is a map of services that have tools and resources registered.
var supportedServices = map[string]struct{}{
	"apps":       {},
	"networking": {},
	"droplets":   {},
	"accounts":   {},
}

func Register(logger *slog.Logger, s *server.MCPServer, c *godo.Client, servicesToActivate ...string) error {
	// There's got to be at least one service.
	if len(servicesToActivate) == 0 {
		return fmt.Errorf("at least one tool must be specified to activate")
	}

	for _, svc := range servicesToActivate {
		if _, ok := supportedServices[svc]; !ok {
			return fmt.Errorf("unsupported tool: %s, supported tools are: %v", svc, supportedServices)
		}

		logger.Debug(fmt.Sprintf("Registering tool and resources for service: %s", svc))

		// register tools for individual services.
		switch svc {
		case "apps":
			s.AddTools()
		case "networking":
			// TODO, existing tools have been categorized into networking. We're probably going to need to categorize them better.
			s.AddTools(tools.NewCDNTool(c).Tools()...)
			s.AddTools(tools.NewCertificateTool(c).Tools()...)
			s.AddTools(tools.NewDomainsTool(c).Tools()...)
			s.AddTools(tools.NewFirewallTool(c).Tools()...)
			s.AddTools(tools.NewKeysTool(c).Tools()...)
			s.AddTools(tools.NewReservedIPTool(c).Tools()...)
			s.AddTools(tools.NewPartnerAttachmentTool(c).Tools()...)
			s.AddTools(tools.NewVPCTool(c).Tools()...)

			// Register the resources for networking
			cdnResource := resources.NewCDNResource(c)
			for template, handler := range cdnResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register certificate resource and resource templates
			certificateResource := resources.NewCertificateMCPResource(c)
			for template, handler := range certificateResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register domains resource
			domainsResource := resources.NewDomainsMCPResource(c)
			for template, handler := range domainsResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register firewall resource
			firewallResource := resources.NewFirewallMCPResource(c)
			for template, handler := range firewallResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register keys resource
			keysResource := resources.NewKeysMCPResource(c)
			for template, handler := range keysResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register regions resource
			regionsResource := resources.NewRegionsMCPResource(c)
			for resource, handler := range regionsResource.Resources() {
				s.AddResource(resource, handler)
			}

			// Register reserved IP resources
			reservedIPResource := resources.NewReservedIPResource(c)
			for template, handler := range reservedIPResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			partnerAttachmentResource := resources.NewPartnerAttachmentMCPResource(c)
			for template, handler := range partnerAttachmentResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

			// Register VPC resource
			vpcResource := resources.NewVPCMCPResource(c)
			for template, handler := range vpcResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

		case "droplets":
			// Register the tools and resources for droplets
			s.AddTools(tools.NewDropletTool(c).Tools()...)

			// Register the resources for droplets
			imageResource := resources.NewImagesMCPResource(c)
			for resource, handler := range imageResource.Resources() {
				s.AddResource(resource, handler)
			}
			for template, handler := range imageResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}
			sizesResource := resources.NewSizesMCPResource(c)
			for resource, handler := range sizesResource.Resources() {
				s.AddResource(resource, handler)
			}
			dropletResource := resources.NewDropletMCPResource(c)
			for template, handler := range dropletResource.ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}

		case "accounts":
			// Register account resource and resource templates
			invoicesResource := resources.NewInvoicesMCPResource(c)
			for resource, handler := range invoicesResource.Resources() {
				s.AddResource(resource, handler)
			}
			for resource, handler := range resources.NewAccountMCPResource(c).Resources() {
				s.AddResource(resource, handler)
			}
			for resource, handler := range resources.NewBalanceMCPResource(c).Resources() {
				s.AddResource(resource, handler)
			}
			for template, handler := range resources.NewBillingMCPResource(c).ResourceTemplates() {
				s.AddResourceTemplate(template, handler)
			}
			// Register action resource
			for template, handler := range resources.NewActionMCPResource(c).Resources() {
				s.AddResourceTemplate(template, handler)
			}
		}
	}

	return nil
}
