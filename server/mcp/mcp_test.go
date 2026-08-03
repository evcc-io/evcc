package mcp

import (
	"encoding/json"
	"net/http"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toolNames lists the tools the given server exposes
func toolNames(t *testing.T, srv *mcpsdk.Server) []string {
	t.Helper()

	ctx := t.Context()

	ct, st := mcpsdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	require.NoError(t, err)
	defer ss.Close()

	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer cs.Close()

	var res []string
	for params := new(mcpsdk.ListToolsParams); ; {
		list, err := cs.ListTools(ctx, params)
		require.NoError(t, err)

		for _, tool := range list.Tools {
			res = append(res, tool.Name)
		}

		if list.NextCursor == "" {
			return res
		}
		params = &mcpsdk.ListToolsParams{Cursor: list.NextCursor}
	}
}

// the state document is large, the tool defaults to listing its keys
// toolSchema returns the input schema a client is offered for a tool
func toolSchema(t *testing.T, srv *mcpsdk.Server, name string) map[string]any {
	t.Helper()

	ctx := t.Context()

	ct, st := mcpsdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	require.NoError(t, err)
	defer ss.Close()

	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer cs.Close()

	for params := new(mcpsdk.ListToolsParams); ; {
		list, err := cs.ListTools(ctx, params)
		require.NoError(t, err)

		for _, tool := range list.Tools {
			if tool.Name == name {
				b, err := json.Marshal(tool.InputSchema)
				require.NoError(t, err)

				var res map[string]any
				require.NoError(t, json.Unmarshal(b, &res))

				return res
			}
		}

		if list.NextCursor == "" {
			return nil
		}
		params = &mcpsdk.ListToolsParams{Cursor: list.NextCursor}
	}
}

func TestGetStateDefaultsToKeys(t *testing.T) {
	srv, err := New(http.NotFoundHandler())
	require.NoError(t, err)

	schema := toolSchema(t, srv, "getState")
	require.NotNil(t, schema)

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "properties")

	jq, ok := props["jq"].(map[string]any)
	require.True(t, ok, "jq parameter")
	assert.Equal(t, "keys", jq["default"])
}

func TestReadOnly(t *testing.T) {
	full, err := New(http.NotFoundHandler())
	require.NoError(t, err)

	ro, err := New(http.NotFoundHandler(), ReadOnly())
	require.NoError(t, err)

	fullNames, roNames := toolNames(t, full), toolNames(t, ro)

	assert.Contains(t, fullNames, "setLoadpointMode")
	assert.Less(t, len(roNames), len(fullNames))

	// reading the system and the documentation stays possible
	for _, name := range []string{"getState", "getTariffInfo", "docs", "fetchDocs"} {
		assert.Contains(t, roNames, name)
	}

	// nothing that changes the system survives
	for _, name := range roNames {
		assert.NotRegexp(t, "^(set|delete|remove|update|assign|start|disable)", name, "not read-only")
	}
}
