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
	var calls int
	status := http.StatusOK

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/optimize/explain", r.URL.Path)
		calls++
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
	assert.Equal(t, 1, calls)

	// same plan and language is served from cache
	text, err = site.ExplainOptimizerResult(context.Background(), "de")
	require.NoError(t, err)
	assert.Equal(t, "because", text)
	assert.Equal(t, 1, calls)

	// another language asks again
	_, err = site.ExplainOptimizerResult(context.Background(), "en")
	require.NoError(t, err)
	assert.Equal(t, 2, calls)

	// new plan asks again, errors are not cached
	site.optimizerResult = &optimizerResult{
		Req: optimizer.OptimizationInput{TimeSeries: optimizer.TimeSeries{Dt: []int{900}}},
		Res: optimizer.OptimizationResult{Status: optimizer.Optimal},
	}

	status = http.StatusServiceUnavailable
	_, err = site.ExplainOptimizerResult(context.Background(), "de")
	assert.EqualError(t, err, "Plan explanations are not configured")
	assert.Equal(t, 3, calls)

	_, err = site.ExplainOptimizerResult(context.Background(), "de")
	assert.EqualError(t, err, "Plan explanations are not configured")
	assert.Equal(t, 4, calls)
}
