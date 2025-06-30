package mcp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/openapi-mcp/pkg/openapi2mcp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:generate go tool openapi https://raw.githubusercontent.com/evcc-io/docs/refs/heads/main/static/rest-api.yaml

//go:embed openapi.json
var spec []byte

func NewHandler(apiUrl, baseUrl, basePath string) (http.Handler, error) {
	log := util.NewLogger("mcp")
	log.INFO.Printf("MCP listening at %s", baseUrl+basePath)

	var doc *openapi3.T
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}

	if err := openapi3.NewLoader().ResolveRefsIn(doc, nil); err != nil {
		return nil, fmt.Errorf("failed resolving spec references: %v", err)
	}

	doc.Servers = []*openapi3.Server{{
		URL:         apiUrl,
		Description: "evcc api",
	}}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)

	srv := server.NewMCPServer("evcc", util.Version,
		server.WithLogging(),
		server.WithToolFilter(toolFilter(log)),
	)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, &openapi2mcp.ToolGenOptions{
		NameFormat: nameFormat(log),
	})

	handler := server.NewStreamableHTTPServer(srv,
		server.WithEndpointPath(basePath),
		server.WithLogger(&stdLogger{log}),
	)

	return handler, nil
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
