package charger

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/require"
)

// statusErr builds the error a plugin returns for the given response
func statusErr(t *testing.T, code int, body string) error {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://vehicle.test", nil)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}

	_, err = request.ReadBody(resp)
	require.Error(t, err)

	return err
}

func TestAsleep(t *testing.T) {
	sleeping := `{"response":{"result":false,"reason":"vehicle is sleeping"}}`

	// TeslaBleHttpProxy sleeping response
	require.ErrorIs(t, asleep(statusErr(t, http.StatusServiceUnavailable, sleeping)), api.ErrAsleep)

	// 503 for any other reason is not asleep
	other := statusErr(t, http.StatusServiceUnavailable, `{"response":{"reason":"vehicle is offline"}}`)
	require.NotErrorIs(t, asleep(other), api.ErrAsleep)

	// non-503 with sleeping body is not asleep
	require.NotErrorIs(t, asleep(statusErr(t, http.StatusInternalServerError, sleeping)), api.ErrAsleep)

	// non-json body and nil error pass through
	require.NotErrorIs(t, asleep(statusErr(t, http.StatusServiceUnavailable, "sleeping")), api.ErrAsleep)
	require.NoError(t, asleep(nil))
}
