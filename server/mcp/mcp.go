package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/openapi-mcp/pkg/openapi2mcp"
	"github.com/getkin/kin-openapi/openapi3"
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
		NameFormat: nameFormat(log),
		TagFilter: []string{
			"General",
			"Home Battery",
			"Loadpoints",
			"Tariffs",
			"Vehicles",
		},
		RequestHandler: requestHandler(host),
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "docs",
		Description: "Documentation",
	}, docsTool)

	srv.AddPrompt(&mcp.Prompt{
		Name:        "create-charge-plan",
		Description: "Create an optimized charge plan for a loadpoint or vehicle",
		Arguments: []*mcp.PromptArgument{
			{Name: "loadpoint", Description: "The loadpoint to create the charge plan for"},
			{Name: "vehicle", Description: "The vehicle to create the charge plan for"},
		},
	}, promptHandler())

	streamHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)

	// Mount handler at the specified basePath
	mux := http.NewServeMux()
	mux.Handle(basePath, streamHandler)
	handler := mux

	return handler, nil
}

func nameFormat(log *util.Logger) func(name string) string {
	return func(name string) string {
		// move method to the end
		parts := strings.Split(name, "_")
		res := strings.Join(parts[:len(parts)-1], "-") + "-" + parts[len(parts)-1]

		res = strings.TrimPrefix(res, "/")
		res = strings.ReplaceAll(res, "/", "-")
		res = strings.ReplaceAll(res, "{", "_")
		res = strings.ReplaceAll(res, "}", "")
		res = strings.ReplaceAll(res, "-_", "_")
		res = strings.ToLower(res)

		// Claude Code has a 64 character limit for tool names
		if len(res) > 64 {
			res = res[:64]
		}

		log.TRACE.Println("adding tool:", res)
		return res
	}
}

func requestHandler(handler http.Handler) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		return resp, nil
	}
}
