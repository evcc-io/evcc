package mcp

import (
	"context"
	"strconv"

	"github.com/evcc-io/evcc/core/site"
	"github.com/mark3labs/mcp-go/mcp"
)

func loadpointsHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		count := len(site.Loadpoints())
		return mcp.NewToolResultText(strconv.Itoa(count)), nil
	}
}
