package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"

	"github.com/evcc-io/evcc/util"
	openapi2mcp "github.com/evcc-io/openapi-mcp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed openapi.json
var spec []byte

func NewHandler(host http.Handler) (http.Handler, error) {
	log := util.NewLogger("mcp")

	var doc *openapi3.T
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}

	if err := openapi3.NewLoader().ResolveRefsIn(doc, nil); err != nil {
		return nil, fmt.Errorf("failed resolving OpenAPI spec references: %v", err)
	}

	// required for the /api path
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
		RequestHandler: requestHandler(log, host),
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "docs",
		Description: "Documentation",
	}, docsTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)

	return handler, nil
}

func requestHandler(log *util.Logger, handler http.Handler) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		if r, err := httputil.DumpRequest(req, true); err == nil {
			log.TRACE.Println(string(r))
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()

		if r, err := httputil.DumpResponse(resp, true); err == nil {
			log.TRACE.Println(string(r))
		}

		return resp, nil
	}
}
