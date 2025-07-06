package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func docsTool(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewResourceLink("https://docs.evcc.io", "evcc-docs", "evcc documentation", "text/html"),
		},
	}, nil
}
