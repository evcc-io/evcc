package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestShorten(t *testing.T) {
	// plain text and short results are handed through untouched
	for _, s := range []string{"soc is 42", "", `{"soc":42}`, `{"lp":[1,2,3]}`} {
		assert.Equal(t, s, shorten(s, false))
	}

	res := shorten(`{"loadpoints":[1,2,3,4,5,6],"soc":42}`, false)
	assert.Contains(t, res, `"soc":42`, "other values are kept")
	assert.Contains(t, res, "3 array elements omitted")
	assert.Contains(t, res, "… 3 more elements")
	assert.NotContains(t, res, "jq")

	// jq capable tools are told how to get the rest
	assert.Contains(t, shorten(`{"a":[1,2,3,4]}`, true), `{"jq": ".loadpoints"}`)
}

func TestShortenNested(t *testing.T) {
	in := `{"vehicles":[{"plans":[1,2,3,4,5]},{"plans":[1,2]},{"x":1},{"y":2},{"z":3}]}`

	res := shorten(in, false)
	assert.Contains(t, res, "4 array elements omitted", "2 outer plus 2 inner")

	// the shortened result is still valid json
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.Split(res, "\n\n")[0]), &doc))

	vehicles, ok := doc["vehicles"].([]any)
	require.True(t, ok)
	require.Len(t, vehicles, maxElements+1) // elements plus marker
	assert.Equal(t, "… 2 more elements", vehicles[maxElements])
}

func TestShortenToolResult(t *testing.T) {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil)

	type filter struct {
		Jq string `json:"jq,omitempty" jsonschema:"filter the state"`
	}

	mcpsdk.AddTool(srv, &mcpsdk.Tool{Name: "getState", Description: "system state"},
		func(_ context.Context, _ *mcpsdk.CallToolRequest, _ filter) (*mcpsdk.CallToolResult, any, error) {
			var sb strings.Builder
			sb.WriteString(`{"sessions":[`)
			for i := range 20 {
				if i > 0 {
					sb.WriteString(",")
				}
				fmt.Fprintf(&sb, `{"id":%d}`, i)
			}
			sb.WriteString("]}")

			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: sb.String()}},
			}, nil, nil
		})

	a, err := newAssistant(t.Context(), &fakeLLM{}, srv)
	require.NoError(t, err)
	defer a.Close()

	require.True(t, a.supportsJq("getState"))
	require.False(t, a.supportsJq("unknown"))

	res := call(t, a, "getState", "{}")
	assert.Contains(t, res, `{"id":0}`)
	assert.NotContains(t, res, `{"id":19}`)
	assert.Contains(t, res, "17 array elements omitted")
	assert.Contains(t, res, "jq")
	assert.Less(t, len(res), 300, "result stays small")
}

func TestTrimHistory(t *testing.T) {
	msg := func(s string) Message { return Message{Role: "user", Content: s} }

	history := []Message{msg("aaaa"), msg("bbbb"), msg("cccc"), msg("dddd")}

	// everything fits
	assert.Len(t, trimHistory(history, 100), 4)

	// oldest messages go first
	assert.Equal(t, []Message{msg("cccc"), msg("dddd")}, trimHistory(history, 9))

	// the current question survives any budget
	assert.Equal(t, []Message{msg("dddd")}, trimHistory(history, 1))
	assert.Len(t, trimHistory(history[3:], 1), 1)
	assert.Empty(t, trimHistory(nil, 10))
}

func TestOfferedTools(t *testing.T) {
	// without getState only the meta tools are offered
	assert.ElementsMatch(t, []string{findToolsName, callToolName}, toolNames(offeredTools(nil)))

	catalog := []llms.Tool{
		{Function: &llms.FunctionDefinition{Name: "setLoadpointMode"}},
		{Function: &llms.FunctionDefinition{Name: "getState"}},
	}

	// getState joins the meta tools, the rest stays searchable
	assert.ElementsMatch(t,
		[]string{findToolsName, callToolName, "getState"},
		toolNames(offeredTools(catalog)),
	)
}

// openapi tools wrap the payload, the truncation has to see through that
func TestShortenWrapped(t *testing.T) {
	// the url carries the jq filter, brackets and all
	const preamble = "HTTP GET http://localhost:7070/api/state?jq=.loadpoints[0]\nStatus: 200\nResponse:\n"

	res := shorten(preamble+`{"loadpoints":[1,2,3,4,5,6]}`, true)
	assert.True(t, strings.HasPrefix(res, preamble), "preamble is kept")
	assert.Contains(t, res, "3 array elements omitted")
	assert.Contains(t, res, "jq")
}

func TestHintEmpty(t *testing.T) {
	filtered := map[string]any{"jq": ".loadpoints[0].state"}

	// a filter that selects nothing is a dead end without a way out
	for _, body := range []string{"null", "[]", "{}"} {
		res := hintEmpty("HTTP GET x\nStatus: 200\nResponse:\n"+body, filtered)
		assert.Contains(t, res, "selected nothing", body)
	}

	// results and unfiltered calls are left alone
	assert.NotContains(t, hintEmpty(`{"soc":42}`, filtered), "selected nothing")
	assert.NotContains(t, hintEmpty("null", map[string]any{}), "selected nothing")
}
