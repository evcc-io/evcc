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

// chatEvent is a single line of the ndjson response. Steps are sent as they happen,
// the result closes the stream. Errors before the first line keep their status code.
type chatEvent struct {
	Step   *Step   `json:"step,omitempty"`
	Result *Result `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
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

		rc := http.NewResponseController(w)

		// model responses outlive the server's default write timeout
		_ = rc.SetWriteDeadline(time.Now().Add(chatTimeout))

		a, err := New(ctx, cfg, srv)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		defer a.Close()

		// the answer takes tool rounds to arrive, the steps are shown while it is worked on
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)

		var streamed bool
		send := func(ev chatEvent) {
			streamed = true
			_ = enc.Encode(ev)
			_ = rc.Flush()
		}

		res, err := a.WithContext(req.Context).WithSteps(func(step Step) {
			send(chatEvent{Step: &step})
		}).Chat(ctx, req.Messages)

		// a model that fails before the first step still gets its status code, afterwards
		// the response is long committed and the error has to go into the stream
		if err != nil {
			if !streamed {
				jsonError(w, http.StatusBadGateway, err)
				return
			}
			send(chatEvent{Error: err.Error()})
			return
		}

		send(chatEvent{Result: &res})
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
