package mcp

import (
	"net/http"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewHandler(site site.API) http.Handler {
	log := util.NewLogger("mcp")

	// Create a new MCP server
	s := server.NewMCPServer(
		"evcc ☀️🚘",
		util.Version,
		server.WithToolCapabilities(true),
	)

	s.AddResource(
		mcp.NewResource(
			"https://docs.evcc.io",
			"docs",
			mcp.WithResourceDescription("evcc documentation"),
		),
		nil, // TODO no handler needed
	)

	s.AddTool(
		mcp.NewTool(
			"site-loadpoints",
			mcp.WithDescription("Number of loadpoints"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		loadpointsHandler(site),
	)

	s.AddResource(
		mcp.NewResource(
			"loadpoints://{loadpoint_id}",
			"loadpoint-status",
			mcp.WithResourceDescription("Loadpoint status information"),
			mcp.WithMIMEType("application/json"),
		),
		loadpointStatusHandler(site),
	)

	ss := server.NewStreamableHTTPServer(s,
		server.WithLogger(&logAdapter{log}),
		server.WithEndpointPath("/"),
	)

	return ss
}
