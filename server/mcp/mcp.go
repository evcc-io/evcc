package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"slices"

	"github.com/evcc-io/evcc/util"
	openapi2mcp "github.com/evcc-io/openapi-mcp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed openapi.json
var spec []byte

// Option configures the server
type Option func(*options)

type options struct {
	readOnly bool
}

// ReadOnly limits the server to tools that only read state. Small models cope
// badly with the full tool set and it keeps the assistant from changing settings.
func ReadOnly() Option {
	return func(o *options) {
		o.readOnly = true
	}
}

// New creates the evcc MCP server. Tool calls are served by host.
func New(host http.Handler, opt ...Option) (*mcp.Server, error) {
	log := util.NewLogger("mcp")

	var o options
	for _, fn := range opt {
		fn(&o)
	}

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

	if o.readOnly {
		ops = slices.DeleteFunc(ops, func(op openapi2mcp.OpenAPIOperation) bool {
			return op.Method != http.MethodGet
		})
	}

	srv := mcp.NewServer(&mcp.Implementation{Name: "evcc", Version: util.Version}, nil)

	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, &openapi2mcp.ToolGenOptions{
		TagFilter: []string{
			"general",
			"state",
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

	mcp.AddTool(srv, &mcp.Tool{
		Name: "fetchDocs",
		Description: "Read a page of the evcc documentation or website as markdown. " +
			"Only urls on evcc.io are allowed. Use this to look up configuration, supported " +
			"devices, plugins or error messages instead of answering from memory. " +
			"Start at https://docs.evcc.io/en for the table of contents, the result lists " +
			"the links of the page to follow. The result is the heading outline plus the cost " +
			"of the entire page, read on with query=<heading> or details=full.",
	}, fetchTool(log))

	return srv, nil
}

// Handler exposes the MCP server via streamable HTTP
func Handler(srv *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)
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
