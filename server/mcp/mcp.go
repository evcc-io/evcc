package mcp

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/evcc-io/evcc/util"
	"github.com/getkin/kin-openapi/openapi3"
	mcpserver "github.com/jedisct1/openapi-mcp/pkg/mcp/server"
	"github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
)

const (
	mcpUri = "https://raw.githubusercontent.com/evcc-io/docs/refs/heads/main/static/rest-api.yaml"
)

func NewHandler(baseUrl, basePath string) (http.Handler, error) {
	uri, _ := url.Parse(mcpUri)

	// set the base URL for OpenAPI spec if not already set
	if os.Getenv("OPENAPI_BASE_URL") == "" {
		os.Setenv("OPENAPI_BASE_URL", baseUrl)
	}

	doc, err := openapi3.NewLoader().LoadFromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)

	var opts []mcpserver.ServerOption
	srv := mcpserver.NewMCPServer("evcc", util.Version, opts...)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, nil)

	streamableServer := mcpserver.NewStreamableHTTPServer(srv,
		// mcpserver.WithHTTPContextFunc(streamableAuthContextFunc),
		mcpserver.WithEndpointPath(basePath),
	)

	return streamableServer, nil
}
