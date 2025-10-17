# Networking MCP Tools

This directory contains tools and resources for managing DigitalOcean networking features via the MCP Server. These tools enable you to create, modify, and query networking resources such as domains, certificates, firewalls, reserved IPs, VPCs, and CDNs.

---

## Supported Tools

### Domains

- **domain-create**
  Create a new domain.
  **Arguments:**
  - `Name` (string, required): Name of the domain
  - `IPAddress` (string, required): IP address for the domain

- **domain-delete**
  Delete a domain.
  - `Name` (string, required): Name of the domain to delete

- **domain-record-create**
  Create a new domain record.
  - `Domain` (string, required): Domain name
  - `Type` (string, required): Record type (e.g., A, CNAME, TXT)
  - `Name` (string, required): Record name
  - `Data` (string, required): Record data

- **domain-record-delete**
  Delete a domain record.
  - `Domain` (string, required): Domain name
  - `RecordID` (number, required): ID of the record to delete

- **domain-record-edit**
  Edit a domain record.
  - `Domain` (string, required): Domain name
  - `RecordID` (number, required): ID of the record to edit
  - `Type` (string, required): Record type
  - `Name` (string, required): Record name
  - `Data` (string, required): Record data

- **domain-get**  
  Get domain information by name.  
  - `Name` (string, required): Name of the domain

- **domain-list**  
  List domains with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

- **domain-record-get**  
  Get a domain record by ID.  
  - `Domain` (string, required): Domain name
  - `RecordID` (number, required): ID of the record

- **domain-record-list**  
  List domain records for a domain with pagination.  
  - `Domain` (string, required): Domain name  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### Certificates

- **custom-certificate-create**
  Create a new custom certificate.
  - `Name` (string, required): Name of the certificate
  - `PrivateKey` (string, required): Private key for the certificate
  - `LeafCertificate` (string, required): Leaf certificate
  - `CertificateChain` (string, required): Certificate chain

- **lets-encrypt-certificate-create**
  Create a new Let's Encrypt certificate.
  - `Name` (string, required): Name of the certificate
  - `DnsNames` (array of strings, required): DNS names of the certificate, including wildcard domains

- **certificate-delete**
  Delete a certificate.
  - `ID` (string, required): ID of the certificate to delete

- **certificate-get**  
  Get certificate information by ID.  
  - `ID` (string, required): ID of the certificate

- **certificate-list**  
  List certificates with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### Firewalls

- **firewall-create**
  Create a new firewall.
  - `Name` (string, required): Name of the firewall
  - `InboundProtocol` (string, required): Protocol for inbound rule
  - `InboundPortRange` (string, required): Port range for inbound rule
  - `InboundSource` (string, required): Source address for inbound rule
  - `OutboundProtocol` (string, required): Protocol for outbound rule
  - `OutboundPortRange` (string, required): Port range for outbound rule
  - `OutboundDestination` (string, required): Destination address for outbound rule
  - `DropletIDs` (array of numbers, optional): Droplet IDs to apply the firewall to
  - `Tags` (array of strings, optional): Tags to apply the firewall to

- **firewall-delete**
  Delete a firewall.
  - `ID` (string, required): ID of the firewall to delete

- **firewall-add-tags**
  Add one or more tags to a firewall.
  - `ID` (string, required): ID of the firewall to update tags
  - `Tags` (array of strings, required): Tags to apply the firewall to

- **firewall-remove-tags**
  Remove one or more tags from a firewall.
  - `ID` (string, required): ID of the firewall to update tags
  - `Tags` (array of strings, required): Tags to remove from the firewall

- **firewall-add-droplets**
  Add one or more droplets to a firewall.
  - `ID` (string, required): ID of the firewall to apply to droplets
  - `DropletIDs` (array of numbers, required): Droplet IDs to apply the firewall to

- **firewall-remove-droplets**
  Remove one or more droplets from a firewall.
  - `ID` (string, required): ID of the firewall to remove droplets from
  - `DropletIDs` (array of numbers, required): Droplet IDs to remove from the firewall

- **firewall-add-rules**
  Add one or more rules to a firewall.
  - `ID` (string, required): ID of the firewall to add rules to
  - `InboundRules` (array of objects, optional): Inbound rules to add
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Sources` (array of strings, required): Source IP addresses or CIDR blocks
  - `OutboundRules` (array of objects, optional): Outbound rules to add
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Destinations` (array of strings, required): Destination IP addresses or CIDR blocks

- **firewall-remove-rules**
  Remove one or more rules from a firewall.
  - `ID` (string, required): ID of the firewall to remove rules from
  - `InboundRules` (array of objects, optional): Inbound rules to remove
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Sources` (array of strings, required): Source IP addresses or CIDR blocks
  - `OutboundRules` (array of objects, optional): Outbound rules to remove
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Destinations` (array of strings, required): Destination IP addresses or CIDR blocks

- **firewall-get**  
  Get firewall information by ID.  
  - `ID` (string, required): ID of the firewall

- **firewall-list**  
  List firewalls with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---


### Reserved IPs

- **reserved-ip-reserve**
  Reserve a new IPv4 or IPv6.
  - `Region` (string, required): Region to reserve the IP in
  - `Type` (string, required): Type of IP to reserve (`ipv4` or `ipv6`)

- **reserved-ip-release**
  Release a reserved IPv4 or IPv6.
  - `IP` (string, required): The reserved IP to release
  - `Type` (string, required): Type of IP to release (`ipv4` or `ipv6`)

- **reserved-ip-assign**
  Assign a reserved IP to a droplet.
  - `IP` (string, required): The reserved IP to assign
  - `DropletID` (number, required): The ID of the droplet
  - `Type` (string, required): Type of IP (`ipv4` or `ipv6`)

- **reserved-ip-unassign**
  Unassign a reserved IP from a droplet.
  - `IP` (string, required): The reserved IP to unassign
  - `Type` (string, required): Type of IP (`ipv4` or `ipv6`)

- **reserved-ip-list**
  List reserved IPv4 addresses with pagination.
  - `Type` (string, required): Type of IP (`ipv4` or `ipv6`)
  - `Page` (number, optional, default: 1): Page number
  - `PerPage` (number, optional, default: 20): Items per page

- **reserved-ip-get**  
  Get reserved IPv4 information by IP.  
  - `IP` (string, required): The reserved IPv4 or IPv6 address

---

### VPC Peering

- **vpc-peering-create**
  Create a new VPC Peering connection between two VPCs.
  - `Name` (string, required): Name for the Peering connection
  - `Vpc1` (string, required): ID of the first VPC
  - `Vpc2` (string, required): ID of the second VPC

- **vpc-peering-delete**
  Delete a VPC Peering connection.
  - `ID` (string, required): ID of the VPC Peering connection to delete

- **vpc-peering-get**  
  Get VPC Peering information by ID.  
  - `ID` (string, required): ID of the VPC Peering connection

- **vpc-peering-list**  
  List VPC Peering connections with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### VPCs

- **vpc-create**
  Create a new VPC.
  - `Name` (string, required): Name of the VPC
  - `Region` (string, required): Region slug (e.g., nyc3)
  - `Subnet` (string, optional): Optional subnet CIDR block (e.g., 10.10.0.0/20)
  - `Description` (string, optional): Optional description for the VPC

- **vpc-list-members**
  List members of a VPC.
  - `ID` (string, required): ID of the VPC

- **vpc-delete**
  Delete a VPC.
  - `ID` (string, required): ID of the VPC to delete

- **vpc-get**  
  Get VPC information by ID.  
  - `ID` (string, required): ID of the VPC

- **vpc-list**  
  List VPCs with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---


## Example Queries Using Networking MCP Tools

- Create a new domain "example.com" pointing to IP "203.0.113.10".
- Add an A record to "example.com" for "www" pointing to "203.0.113.20".
- Delete the TXT record with ID 12345 from "example.com".
- Create a new custom SSL certificate for "myapp.com".
- Create a new Let's Encrypt certificate for "example.com" and "www.example.com".
- Create a wildcard Let's Encrypt certificate for "*.example.com" and "example.com".
- Delete a firewall with ID "abcd-1234".
- Add HTTP and HTTPS inbound rules to firewall "fw-123".
- Remove SSH access rule from firewall "fw-456".
- Reserve a new IPv4 in region "nyc3".
- Assign reserved IP "198.51.100.5" to droplet 987654.
- Create a new VPC named "private-net" in region "sfo2".
- Flush the cache for CDN with ID "cdn-xyz" for file "/static/logo.png".

---

## Notes

- All resource identifiers (IDs, names, IPs) must be replaced with actual values in your queries.
- All responses are returned in JSON format for easy parsing and integration.
- For endpoints that require an ID, name, or IP, replace the placeholder with the appropriate value.
- Use the tools to automate and manage all aspects of networking from domains and DNS to VPCs, firewalls, and advanced partner connectivity.

---
