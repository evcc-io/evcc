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
	mcptypes "github.com/jedisct1/openapi-mcp/pkg/mcp/mcp"
	mcpserver "github.com/jedisct1/openapi-mcp/pkg/mcp/server"
	"github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
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

	opts := []mcpserver.ServerOption{
		mcpserver.WithLogging(),
		mcpserver.WithHooks(&mcpserver.Hooks{
			OnAfterListTools: []mcpserver.OnAfterListToolsFunc{toolFilter(log)},
		}),
	}

	srv := mcpserver.NewMCPServer("evcc", util.Version, opts...)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, &openapi2mcp.ToolGenOptions{
		NameFormat: nameFormat(log),
	})

	streamableServer := mcpserver.NewStreamableHTTPServer(srv,
		// mcpserver.WithHTTPContextFunc(streamableAuthContextFunc),
		mcpserver.WithEndpointPath(basePath),
		mcpserver.WithLogger(&stdLogger{log}),
	)

	return streamableServer, nil
}

func nameFormat(log *util.Logger) func(name string) string {
	return func(name string) string {
		res := name
		res = strings.ReplaceAll(res, "_/", "/")
		res = strings.ReplaceAll(res, "/", "-")
		res = strings.ReplaceAll(res, "{", "")
		res = strings.ReplaceAll(res, "}", "")
		res = strings.ToLower(res)
		log.TRACE.Println("adding tool:", res)
		return res
	}
}

func toolFilter(log *util.Logger) func(ctx context.Context, id any, message *mcptypes.ListToolsRequest, result *mcptypes.ListToolsResult) {
	blocked := []string{"auth", "config", "system"}

	return func(ctx context.Context, id any, message *mcptypes.ListToolsRequest, result *mcptypes.ListToolsResult) {
		var res []mcptypes.Tool

	TOOLS:
		for _, tool := range result.Tools {
			for _, s := range blocked {
				if strings.Contains(tool.Name, s) {
					log.TRACE.Println("skipping tool:", tool.Name)
					continue TOOLS
				}
			}

			res = append(res, tool)
		}

		result.Tools = res
	}
}
