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
	_ = log

	// Create a new MCP server
	s := server.NewMCPServer(
		"evcc ☀️🚘",
		util.Version,
		server.WithToolCapabilities(false),
	)

	s.AddResource(
		mcp.NewResource(
			"https://docs.evcc.io",
			"docs",
			mcp.WithResourceDescription("evcc documentation"),
		),
		nil, // TODO no handler needed
	)

	// // Start the stdio server
	// if err := server.ServeStdio(s); err != nil {
	// 	log.ERROR.Println("cannot start server:", err)
	// }

	// return server.NewStreamableHTTPServer(s, server.WithLogger(&logAdapter{log}))
	sse := server.NewSSEServer(s, server.WithStaticBasePath("/mcp"))
	return sse.SSEHandler()
}
