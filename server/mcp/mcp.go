package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/evcc-io/evcc/util"
	openapi2mcp "github.com/evcc-io/openapi-mcp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed openapi.json
var spec []byte

func NewHandler(host http.Handler, baseUrl, basePath string) (http.Handler, error) {
	log := util.NewLogger("mcp")
	log.INFO.Printf("MCP listening at %s", baseUrl+basePath)

	var doc *openapi3.T
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}

	if err := openapi3.NewLoader().ResolveRefsIn(doc, nil); err != nil {
		return nil, fmt.Errorf("failed resolving OpenAPI spec references: %v", err)
	}

	doc.Servers = []*openapi3.Server{{
		URL:         "http://localhost:7070/api",
		Description: "evcc api",
	}}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)

	srv := mcp.NewServer(&mcp.Implementation{Name: "evcc", Version: util.Version}, nil)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, &openapi2mcp.ToolGenOptions{
		TagFilter: []string{
			"general",
			"tariffs",
			"loadpoints",
			"vehicles",
			"battery",
		},
		RequestHandler: requestHandler(host),
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "docs",
		Description: "Documentation",
		InputSchema: emptySchema(),
	}, docsTool)

	srv.AddPrompt(&mcp.Prompt{
		Name:        "create-charge-plan",
		Description: "Create an optimized charge plan for a loadpoint or vehicle",
		Arguments: []*mcp.PromptArgument{
			{Name: "loadpoint", Description: "The loadpoint to create the charge plan for"},
			{Name: "vehicle", Description: "The vehicle to create the charge plan for"},
		},
	}, promptHandler())

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)

	return handler, nil
}

func requestHandler(handler http.Handler) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		return resp, nil
	}
}

func emptySchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
}
