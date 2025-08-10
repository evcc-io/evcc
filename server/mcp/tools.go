package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func docsTool(_ context.Context, _ *mcp.ServerSession, _ *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.ResourceLink{
				URI:      "https://docs.evcc.io",
				Name:     "evcc-docs",
				Title:    "evcc documentation",
				MIMEType: "text/html",
			},
		},
	}, nil
}
