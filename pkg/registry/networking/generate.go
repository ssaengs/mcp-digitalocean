package networking

//go:generate mockgen -destination=./mocks.go -package networking github.com/digitalocean/godo  CertificatesService,DomainsService,FirewallsService,PartnerAttachmentService,ReservedIPsService,ReservedIPV6sService,ReservedIPActionsService,ReservedIPV6ActionsService,VPCsService
