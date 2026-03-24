package transport

import (
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
)

func BrotliCompression(base http.RoundTripper) http.RoundTripper {
	return &brotliTransport{base: base}
}

type brotliTransport struct {
	base http.RoundTripper
}

func (t *brotliTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	// Clone request so we don't mutate caller's headers
	r := req.Clone(req.Context())
	if existing := r.Header.Get("Accept-Encoding"); existing == "" {
		r.Header.Set("Accept-Encoding", "br")
	} else {
		r.Header.Set("Accept-Encoding", existing+", br")
	}

	resp, err := base.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Encoding") == "br" {
		resp.Body = &brotliReadCloser{
			Reader: brotli.NewReader(resp.Body),
			Closer: resp.Body,
		}

		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = -1
	}

	return resp, nil
}

type brotliReadCloser struct {
	io.Reader
	io.Closer
}
