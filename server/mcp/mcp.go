package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	mcpUri = "https://raw.githubusercontent.com/evcc-io/docs/refs/heads/main/static/rest-api.yaml"
)

func NewHandler(apiUrl, baseUrl, basePath string) (http.Handler, error) {
	uri, err := url.Parse(mcpUri)
	if err != nil {
		return nil, err
	}

	// set the base URL for OpenAPI spec if not already set
	if os.Getenv("OPENAPI_BASE_URL") == "" {
		os.Setenv("OPENAPI_BASE_URL", apiUrl)
	}

	log := util.NewLogger("mcp")
	log.INFO.Printf("MCP listening at %s", baseUrl+basePath)

	doc, err := openapi3.NewLoader().LoadFromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)

	opts := []server.ServerOption{
		server.WithLogging(),
		// server.WithHooks(&server.Hooks{
		// 	OnAfterListTools: []server.OnAfterListToolsFunc{requestToolFilter(log)},
		// }),
		server.WithToolFilter(toolFilter(log)),
	}

	srv := server.NewMCPServer("evcc", util.Version, opts...)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, &openapi2mcp.ToolGenOptions{
		NameFormat: nameFormat(log),
	})

	streamableServer := server.NewStreamableHTTPServer(srv,
		// server.WithHTTPContextFunc(streamableAuthContextFunc),
		server.WithEndpointPath(basePath),
		server.WithLogger(&stdLogger{log}),
	)

	return streamableServer, nil
}

func nameFormat(log *util.Logger) func(name string) string {
	return func(name string) string {
		res := name
		res = strings.ReplaceAll(res, "_/", "/")
		res = strings.ReplaceAll(res, "/", "-")
		res = strings.ReplaceAll(res, "{", "with_")
		res = strings.ReplaceAll(res, "}", "")
		res = strings.ToLower(res)
		log.TRACE.Println("adding tool:", res)
		return res
	}
}

func filterTools(log *util.Logger, tools []mcp.Tool) []mcp.Tool {
	var res []mcp.Tool

TOOLS:
	for _, tool := range tools {
		for _, block := range []string{"auth", "config", "system"} {
			if strings.Contains(tool.Name, block) {
				log.TRACE.Println("skipping tool:", tool.Name)
				continue TOOLS
			}
		}

		res = append(res, tool)
	}

	return res
}

func toolFilter(log *util.Logger) server.ToolFilterFunc {
	return func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		return filterTools(log, tools)
	}
}

// func requestToolFilter(log *util.Logger) func(ctx context.Context, id any, message *mcp.ListToolsRequest, result *mcp.ListToolsResult) {
// 	return func(ctx context.Context, id any, message *mcp.ListToolsRequest, result *mcp.ListToolsResult) {
// 		result.Tools = filterTools(log, result.Tools)
// 	}
// }
