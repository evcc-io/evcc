package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgo "github.com/modelcontextprotocol/go-sdk/mcp"
)

func docsTool(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewResourceLink("https://docs.evcc.io", "evcc-docs", "evcc documentation", "text/html"),
		},
	}, nil
}

func docsTool2(_ context.Context, cc *mcpgo.ServerSession, params *mcpgo.CallToolParamsFor[map[string]any]) (*mcpgo.CallToolResultFor[any], error) {
	return &mcpgo.CallToolResultFor[any]{
		Content: []mcpgo.Content{
			&mcpgo.ResourceLink{
				URI:      "https://docs.evcc.io",
				Name:     "evcc-docs",
				Title:    "evcc documentation",
				MIMEType: "text/html",
			},
		},
	}, nil
}
