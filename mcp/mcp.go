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
		server.WithResourceCapabilities(false, true),
		server.WithHooks(hooks(log)),
		server.WithLogging(),
	)

	// tools

	s.AddTool(
		mcp.NewTool(
			"list-loadpoints",
			mcp.WithDescription("List loadpoints"),
			mcp.WithReadOnlyHintAnnotation(true),
			// mcp.WithNumber("id", mcp.Min(0), mcp.Max(float64(len(site.Loadpoints())-1)), mcp.MultipleOf(1)),
			// mcp.WithString("title", mcp.Enum(lo.Map(site.Loadpoints(), func(lp loadpoint.API, _ int) string {
			// 	return lp.GetTitle()
			// })...)),
		),
		siteAllLoadpointsHandler(site),
	)

	s.AddTool(
		mcp.NewTool(
			"solar-forecast",
			mcp.WithDescription("Solar forecast for remaining production today"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		solarForecastHandler(site),
	)

	// resource tools

	s.AddTool(
		mcp.NewTool(
			"list-loadpoint-resources",
			mcp.WithDescription("List loadpoints as resources"),
			mcp.WithReadOnlyHintAnnotation(true),
			// mcp.WithNumber("id", mcp.Min(0), mcp.Max(float64(len(site.Loadpoints())-1)), mcp.MultipleOf(1)),
			// mcp.WithString("title", mcp.Enum(lo.Map(site.Loadpoints(), func(lp loadpoint.API, _ int) string {
			// 	return lp.GetTitle()
			// })...)),
		),
		siteAllLoadpointsAsRessourcesHandler(site),
	)

	// resources

	s.AddResource(
		mcp.NewResource(
			"https://docs.evcc.io",
			"docs",
			mcp.WithResourceDescription("evcc documentation"),
		),
		httpHandler("https://docs.evcc.io"), // TODO no handler needed
	)

	s.AddResource(
		mcp.NewResource(
			"site://loadpoints",
			"site-loadpoints",
			mcp.WithResourceDescription("Get loadpoints"),
			// mcp.WithMIMEType("application/json"),
		),
		allLoadpointsHandler(site),
	)

	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"loadpoint://{id}",
			"loadpoint-status",
			mcp.WithTemplateDescription("Get loadpoint status information"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		loadpointStatusHandler(site),
	)

	ss := server.NewStreamableHTTPServer(s,
		server.WithLogger(&logAdapter{log}),
		server.WithEndpointPath("/"),
	)

	return ss
}
