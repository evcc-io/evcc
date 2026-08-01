package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// docsSite is the only host tree the fetch tool is allowed to access
	docsSite = "evcc.io"

	// maxFetchSize limits the downloaded document
	maxFetchSize = 4 << 20

	// maxFetchText limits the text handed to the model, keeping its context usable
	maxFetchText = 40000

	// maxFetchLinks limits the links offered for further navigation
	maxFetchLinks = 100
)

type fetchParams struct {
	Url string `json:"url" jsonschema:"URL of the page to read, must be on evcc.io"`
}

// fetchTool reads a documentation page and returns it as plain text
func fetchTool(log *util.Logger) func(context.Context, *mcp.CallToolRequest, fetchParams) (*mcp.CallToolResult, any, error) {
	client := request.NewHelper(log)

	// redirects must not leave the allowed site
	client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		return validateUrl(req.URL)
	}

	return func(ctx context.Context, _ *mcp.CallToolRequest, params fetchParams) (*mcp.CallToolResult, any, error) {
		uri, err := url.Parse(strings.TrimSpace(params.Url))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid url: %w", err)
		}
		if err := validateUrl(uri); err != nil {
			return nil, nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
		if err != nil {
			return nil, nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		if err := request.ResponseError(resp); err != nil {
			return nil, nil, err
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchSize))
		if err != nil {
			return nil, nil, err
		}

		text, err := documentText(uri, body, resp.Header.Get("Content-Type"))
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}

// validateUrl ensures the url points at the evcc website or documentation
func validateUrl(uri *url.URL) error {
	if uri.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", uri.Scheme)
	}

	if host := uri.Hostname(); host != docsSite && !strings.HasSuffix(host, "."+docsSite) {
		return fmt.Errorf("host not allowed: %s", host)
	}

	return nil
}

// documentText reduces a html document to its readable text and the links to follow.
// Other content types are returned as-is.
func documentText(uri *url.URL, body []byte, contentType string) (string, error) {
	if !strings.Contains(contentType, "text/html") {
		return truncate(string(body)), nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	// collect before stripping navigation, that's where the table of contents lives
	links := documentLinks(uri, doc)

	doc.Find("script, style, noscript, nav, header, footer").Remove()

	content := doc.Find("main")
	if content.Length() == 0 {
		content = doc.Find("body")
	}

	// text nodes are concatenated without separator, keep block elements apart
	content.Find("h1, h2, h3, h4, h5, h6, p, li, tr, br, div, pre, blockquote").AfterHtml("\n")

	var sb strings.Builder
	for line := range strings.SplitSeq(content.Text(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			sb.WriteString(line + "\n")
		}
	}

	res := truncate(sb.String())
	if len(links) > 0 {
		res += "\nLinks on this page:\n" + strings.Join(links, "\n")
	}

	return res, nil
}

// documentLinks returns the unique links of the document that are fetchable as well
func documentLinks(uri *url.URL, doc *goquery.Document) []string {
	var res []string
	seen := make(map[string]bool)

	doc.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")

		link, err := uri.Parse(href)
		if err != nil || validateUrl(link) != nil {
			return true
		}

		link.Fragment = ""
		if s := link.String(); !seen[s] {
			seen[s] = true
			res = append(res, s)
		}

		return len(res) < maxFetchLinks
	})

	return res
}

func truncate(s string) string {
	if len(s) <= maxFetchText {
		return s
	}
	return s[:maxFetchText] + "\n[truncated, fetch a more specific page for the full text]"
}
