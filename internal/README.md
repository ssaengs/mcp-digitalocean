# Onboarding your tools

To onboard a new service you'll need to do the following:

1. **Create a new service directory**: Create a new directory under `internal/` with the name of your service.
2. **Implement the tools** Within the service directory. 
3. **Update `registry.go`** Add your service to `supportedServices` and update the register function to include your service's tools.
4. **Update the README**: Document your service and its tools in the `README.md` file within your service directory.
5. **Create a PR**: Submit a pull request with your changes.

# Debugging and Troubleshooting

During development, you may need to run the mcpServer locally with your favorite IDE. To do this, you can use the following command:

```bash
# build the MCP server
make build-dist
```

Update your IDE mcp server configuration to use the local version:
```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["/PATH/TO/PROJECT/mcp-digitalocean/scripts/npm", "--services apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": ""
      }
    }
  }
}
```
