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
		server.WithResourceCapabilities(true, true),
		server.WithHooks(hooks(log)),
		server.WithLogging(),
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

	s.AddTool(
		mcp.NewTool(
			"solar-forecast",
			mcp.WithDescription("Solar forecast for remaining production today"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		solarForecastHandler(site),
	)

	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"loadpoint://{id}",
			"loadpoint-status",
			mcp.WithTemplateDescription("Loadpoint status information, id starting at 1 for first loadpoint"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		loadpointStatusHandler(log, site),
	)

	ss := server.NewStreamableHTTPServer(s,
		server.WithLogger(&logAdapter{log}),
		server.WithEndpointPath("/"),
	)

	return ss
}
