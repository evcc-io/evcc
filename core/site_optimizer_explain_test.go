package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainOptimizerResult(t *testing.T) {
	var got explainRequest
	var auth string
	status := http.StatusOK

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/optimize/explain", r.URL.Path)
		auth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))

		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"explanation":"because"}`))
		} else {
			_, _ = w.Write([]byte(`{"message":"Plan explanations are not configured"}`))
		}
	}))
	defer srv.Close()

	t.Setenv("OPTIMIZER_URI", srv.URL)
	sponsor.Subject, sponsor.Token = "sponsor", "token"
	t.Cleanup(func() { sponsor.Subject, sponsor.Token = "", "" })

	site := &Site{log: util.NewLogger("foo")}

	_, err := site.ExplainOptimizerResult(context.Background(), "de")
	assert.EqualError(t, err, "no optimizer result")

	site.optimizerResult = &optimizerResult{
		Req: optimizer.OptimizationInput{TimeSeries: optimizer.TimeSeries{Dt: []int{3600}}},
		Res: optimizer.OptimizationResult{Status: optimizer.Optimal},
	}

	text, err := site.ExplainOptimizerResult(context.Background(), "de")
	require.NoError(t, err)
	assert.Equal(t, "because", text)
	assert.Equal(t, "Bearer token", auth)
	assert.Equal(t, "de", got.Language)
	assert.Equal(t, []int{3600}, got.Req.TimeSeries.Dt)
	assert.Equal(t, optimizer.Optimal, got.Res.Status)

	status = http.StatusServiceUnavailable
	_, err = site.ExplainOptimizerResult(context.Background(), "de")
	assert.EqualError(t, err, "Plan explanations are not configured")
}
