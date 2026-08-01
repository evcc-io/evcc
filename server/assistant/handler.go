package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/server/mcp"
	"github.com/evcc-io/evcc/util"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// chatTimeout bounds a single question including all tool calls
const chatTimeout = 5 * time.Minute

type chatRequest struct {
	Messages []Message `json:"messages"`
	Context  string    `json:"context,omitempty"`
}

type chatResponse struct {
	Content string `json:"content"`
}

// ChatHandler answers questions using the configured model and evcc's own MCP tools.
// The MCP server is built lazily and reused, host serves the internal api requests.
func ChatHandler(host http.Handler) http.HandlerFunc {
	server := sync.OnceValues(func() (*mcpsdk.Server, error) {
		// the full set is searchable, only the meta tools are offered to the model
		return mcp.New(host)
	})

	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := ConfiguredConfig()
		if err != nil {
			jsonError(w, http.StatusPreconditionFailed, err)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if len(req.Messages) == 0 {
			jsonError(w, http.StatusBadRequest, errors.New("missing messages"))
			return
		}

		srv, err := server()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), chatTimeout)
		defer cancel()

		// model responses outlive the server's default write timeout
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(chatTimeout))

		a, err := New(ctx, cfg, srv)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		defer a.Close()

		res, err := a.WithContext(req.Context).Chat(ctx, req.Messages)
		if err != nil {
			jsonError(w, http.StatusBadGateway, err)
			return
		}

		jsonWrite(w, chatResponse{Content: res})
	}
}

func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
