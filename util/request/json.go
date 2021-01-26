package request

import (
	"bytes"
	"encoding/json"
	"io"
)

// errorReader wraps an error with an io.Reader
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

// MarshalJSON marshals JSON into an io.Reader
func MarshalJSON(data interface{}) io.Reader {
	if data == nil {
		return nil
	}

	body, err := json.Marshal(data)
	if err != nil {
		return &errorReader{err: err}
	}

	return bytes.NewReader(body)
}
