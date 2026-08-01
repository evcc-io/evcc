package mcp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUrl(t *testing.T) {
	for _, tc := range []struct {
		uri string
		ok  bool
	}{
		{"https://evcc.io/", true},
		{"https://docs.evcc.io/en/faq", true},
		{"https://docs.evcc.io/llms.txt", true},
		{"http://docs.evcc.io/en/faq", false},  // plain http
		{"file:///etc/passwd", false},          // local file
		{"https://notevcc.io/", false},         // suffix without separator
		{"https://evcc.io.evil.com/", false},   // prefix of another domain
		{"https://evil.com/?u=evcc.io", false}, // host in query
		{"https://127.0.0.1/", false},
	} {
		uri, err := url.Parse(tc.uri)
		require.NoError(t, err)

		if err := validateUrl(uri); tc.ok {
			assert.NoError(t, err, tc.uri)
		} else {
			assert.Error(t, err, tc.uri)
		}
	}
}

func TestDocumentText(t *testing.T) {
	uri, err := url.Parse("https://docs.evcc.io/en/faq")
	require.NoError(t, err)

	html := `<html><head><title>t</title><style>body{}</style></head><body>
		<nav><a href="/en/reference">Reference</a></nav>
		<main><h1>Charging</h1><p>Mode is set per loadpoint.</p>
		<a href="https://github.com/evcc-io/evcc">source</a>
		<a href="../installation">install</a></main>
		<footer>imprint</footer></body></html>`

	res, err := documentText(uri, []byte(html), "text/html; charset=utf-8")
	require.NoError(t, err)

	assert.Contains(t, res, "Charging")
	assert.Contains(t, res, "Mode is set per loadpoint.")
	assert.NotContains(t, res, "imprint", "footer should be stripped")
	assert.NotContains(t, res, "body{}", "style should be stripped")

	// navigation links survive stripping, foreign links are dropped
	assert.Contains(t, res, "https://docs.evcc.io/en/reference")
	assert.Contains(t, res, "https://docs.evcc.io/installation")
	assert.NotContains(t, res, "github.com")
}

func TestDocumentTextPlain(t *testing.T) {
	uri, err := url.Parse("https://docs.evcc.io/llms.txt")
	require.NoError(t, err)

	res, err := documentText(uri, []byte("# evcc\n\n> docs"), "text/plain; charset=utf-8")
	require.NoError(t, err)
	assert.Equal(t, "# evcc\n\n> docs", res)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short"))

	res := truncate(strings.Repeat("x", maxFetchText+1))
	assert.True(t, strings.HasPrefix(res, strings.Repeat("x", maxFetchText)))
	assert.Contains(t, res, "[truncated")
}
