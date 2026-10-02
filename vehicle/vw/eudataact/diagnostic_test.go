package eudataact

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type diagnosticRoundTripFunc func(*http.Request) (*http.Response, error)

func (f diagnosticRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLatestDataset(t *testing.T) {
	var requestedDataset string
	archive := testDatasetArchive(t)
	api := &API{Helper: &request.Helper{Client: &http.Client{
		Transport: diagnosticRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			var body io.Reader
			switch req.URL.Path {
			case "/proxy_api/euda-apim/datarequest/vehicles/test-vin/metadata/partial":
				body = strings.NewReader(`{"Identifier":"test-id"}`)
			case "/proxy_api/euda-apim/datadelivery/vehicles/test-vin/test-id/list":
				body = strings.NewReader(`[
					{"name":"older.zip","createdOn":"2026-05-31T08:00:00Z"},
					{"name":"newest.zip","createdOn":"2026-05-31T09:00:00Z"},
					{"name":"20260531100000_no_content_found.zip","createdOn":"2026-05-31T10:00:00Z"}
				]`)
			case "/proxy_api/euda-apim/datadelivery/vehicles/test-vin/test-id/download":
				requestedDataset = req.Header.Get("filename")
				body = bytes.NewReader(archive)
			default:
				t.Fatalf("unexpected request: %s", req.URL)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(body),
				Request:    req,
			}, nil
		}),
	}}}

	got, err := api.LatestDataset("test-vin")
	require.NoError(t, err)
	assert.Equal(t, "newest.zip", got.Name)
	assert.Equal(t, 2, got.PointCount)
	assert.Equal(t, "newest.zip", requestedDataset)
}

func testDatasetArchive(t *testing.T) []byte {
	t.Helper()

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("dataset.json")
	require.NoError(t, err)
	_, err = file.Write([]byte(`{"Data":[
		{"dataFieldName":"state_of_charge","value":"70"},
		{"dataFieldName":"mileage","value":"100"}
	]}`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return archive.Bytes()
}
