package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func docsTool(_ context.Context, _ *mcpsdk.ServerSession, _ *mcpsdk.CallToolParams) (*mcpsdk.CallToolResult, error) {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.ResourceLink{
				URI:      "https://docs.evcc.io",
				Name:     "evcc-docs",
				Title:    "evcc documentation",
				MIMEType: "text/html",
			},
		},
	}, nil
}
