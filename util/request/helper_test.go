package request

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHelper(status int, body string) *Helper {
	return &Helper{
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
}

func TestDoJSON(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	t.Run("empty body is not an error", func(t *testing.T) {
		var res struct{ Code int }
		err := testHelper(http.StatusOK, "").DoJSON(req, &res)
		assert.NoError(t, err)
	})

	t.Run("valid body is decoded", func(t *testing.T) {
		var res struct{ Code int }
		err := testHelper(http.StatusOK, `{"Code":528}`).DoJSON(req, &res)
		assert.NoError(t, err)
		assert.Equal(t, 528, res.Code)
	})

	t.Run("truncated body is an error", func(t *testing.T) {
		var res struct{ Code int }
		err := testHelper(http.StatusOK, `{"Code":52`).DoJSON(req, &res)
		assert.Error(t, err)
	})
}
