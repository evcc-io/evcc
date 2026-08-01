package mcp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPage = `<html><head><title>t</title><style>body{}</style></head><body>
	<nav><a href="/en/reference">Reference</a></nav>
	<main><h1>Charging</h1><p>Mode is set per loadpoint.</p>
	<h2>Modes</h2><p>Four modes exist.</p>
	<h3>Solar</h3><p>Charges on surplus.</p>
	<h2>Phases</h2><p>One or three.</p>
	<h4>Details</h4><p>Too deep for the summary.</p>
	<pre><code class="language-yaml"># not a heading
mode: pv</code></pre>
	<a href="https://github.com/evcc-io/evcc">source</a>
	<a href="../installation">install</a></main>
	<footer>imprint</footer></body></html>`

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

func testDocument(t *testing.T, params fetchParams) string {
	t.Helper()

	uri, err := url.Parse("https://docs.evcc.io/en/faq")
	require.NoError(t, err)

	res, err := documentText(uri, []byte(testPage), "text/html; charset=utf-8", params)
	require.NoError(t, err)

	return res
}

func TestDocumentFull(t *testing.T) {
	res := testDocument(t, fetchParams{Details: detailsFull})

	assert.Contains(t, res, "# Charging")
	assert.Contains(t, res, "Mode is set per loadpoint.")
	assert.Contains(t, res, "```yaml", "code blocks stay fenced")
	assert.NotContains(t, res, "imprint", "footer should be stripped")
	assert.NotContains(t, res, "body{}", "style should be stripped")

	// content links are made absolute, foreign links keep their text only
	assert.Contains(t, res, "[install](https://docs.evcc.io/installation)")
	assert.Contains(t, res, "source")
	assert.NotContains(t, res, "github.com")

	// the site navigation repeats on every page and is not listed
	assert.NotContains(t, res, "https://docs.evcc.io/en/reference")
	assert.Contains(t, res, "Links on this page:\nhttps://docs.evcc.io/installation")
}

func TestDocumentSummary(t *testing.T) {
	res := testDocument(t, fetchParams{})

	assert.Contains(t, res, "# Charging")
	assert.Contains(t, res, "## Modes")
	assert.Contains(t, res, "### Solar")
	assert.NotContains(t, res, "#### Details", "level 4 is below the summary level")
	assert.NotContains(t, res, "Four modes exist.", "body text is left out")
	assert.NotContains(t, res, "not a heading", "yaml comments are not headings")
	assert.Contains(t, res, "summary of 4 headings")
	assert.Contains(t, res, "tokens", "the cost of the entire page is stated")
}

func TestDocumentQuery(t *testing.T) {
	res := testDocument(t, fetchParams{Query: "modes"})

	assert.Contains(t, res, "## Modes")
	assert.Contains(t, res, "Four modes exist.")
	assert.Contains(t, res, "### Solar", "subsections are included")
	assert.Contains(t, res, "Charges on surplus.")
	assert.NotContains(t, res, "## Phases", "the next heading of equal level ends the section")
	assert.NotContains(t, res, "Mode is set per loadpoint.", "preceding sections are left out")

	// unknown heading falls back to the summary
	res = testDocument(t, fetchParams{Query: "nonexistent"})
	assert.Contains(t, res, `no heading matches "nonexistent"`)
	assert.Contains(t, res, "## Modes")
}

func TestDocumentRaw(t *testing.T) {
	assert.Equal(t, testPage, testDocument(t, fetchParams{Mode: modeRaw}))
}

func TestDocumentTextPlain(t *testing.T) {
	uri, err := url.Parse("https://docs.evcc.io/llms.txt")
	require.NoError(t, err)

	res, err := documentText(uri, []byte("# evcc\n\n> docs"), "text/plain; charset=utf-8", fetchParams{})
	require.NoError(t, err)
	assert.Equal(t, "# evcc\n\n> docs", res)
}

func TestSplitMarkdown(t *testing.T) {
	lines, levels := splitMarkdown("# A\ntext\n```yaml\n# comment\n```\n## B\n####### too deep\n#nospace")

	require.Len(t, levels, len(lines))
	assert.Equal(t, []int{1, 0, 0, 0, 0, 2, 0, 0}, levels)
}

func TestValidateParams(t *testing.T) {
	assert.NoError(t, fetchParams{}.validate())
	assert.NoError(t, fetchParams{Mode: modeRaw, Details: detailsFull}.validate())
	assert.Error(t, fetchParams{Mode: "html"}.validate())
	assert.Error(t, fetchParams{Details: "brief"}.validate())
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short"))

	res := truncate(strings.Repeat("x", maxFetchText+1))
	assert.True(t, strings.HasPrefix(res, strings.Repeat("x", maxFetchText)))
	assert.Contains(t, res, "[truncated")
}
