package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sn":"ABC","sites":[{"id":"s1","name":"Home","kw":9.9},{"id":"s2","name":null},{"name":"no id"}]}`))
	}))
	defer srv.Close()

	tc := []struct {
		name   string
		params url.Values
		status int
		res    []service.Option
	}{
		{"list of scalars", url.Values{"jq": {"[.sites[].id]"}}, 200, []service.Option{
			{Value: "s1", Match: true}, {Value: "s2", Match: true},
		}},
		{"single scalar", url.Values{"jq": {".sn"}}, 200, []service.Option{
			{Value: "ABC", Match: true},
		}},
		{"label and hint", url.Values{"jq": {".sites"}, "value": {".id"}, "label": {".name"}, "hint": {".kw"}}, 200, []service.Option{
			{Value: "s1", Label: "Home", Hint: "9.9", Match: true}, {Value: "s2", Match: true},
		}},
		{"jq error", url.Values{"jq": {".missing[]"}}, 200, []service.Option{}},
		{"invalid jq", url.Values{"jq": {"["}}, 400, nil},
		{"invalid label", url.Values{"label": {"["}}, 400, nil},
		{"missing uri", url.Values{"uri": {""}}, 400, nil},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.params["uri"]; !ok {
				tc.params.Set("uri", srv.URL)
			}

			w := httptest.NewRecorder()
			httpGet(w, httptest.NewRequest("GET", "/get?"+tc.params.Encode(), nil))

			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.res == nil {
				return
			}

			var res []service.Option
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			assert.Equal(t, tc.res, res)
		})
	}
}
