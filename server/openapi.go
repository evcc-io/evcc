package server

//go:generate go tool openapi openapi.yaml mcp/openapi.json
//go:generate go tool openapi-mcp --doc mcp/openapi.md openapi.yaml
