package request

import (
	"bytes"
	"encoding/xml"
	"io"
)

// MarshalXML marshals XML into an io.ReadSeeker
func MarshalXML(data any) io.ReadSeeker {
	if data == nil {
		return nil
	}

	body, err := xml.Marshal(data)
	if err != nil {
		return &errorReader{err: err}
	}

	return bytes.NewReader(body)
}
