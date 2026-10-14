package mcp

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
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

	// maxSummaryLevel is the deepest heading level a summary lists
	maxSummaryLevel = 3

	modeMarkdown   = "markdown"
	modeRaw        = "raw"
	detailsSummary = "summary"
	detailsFull    = "full"
)

type fetchParams struct {
	Url     string `json:"url" jsonschema:"URL of the page to read, must be on evcc.io"`
	Mode    string `json:"mode,omitempty" jsonschema:"markdown (default) converts the page to markdown, raw returns the document unchanged"`
	Details string `json:"details,omitempty" jsonschema:"summary (default) lists the headings only, full returns the entire page"`
	Query   string `json:"query,omitempty" jsonschema:"heading to return with its subsections, falls back to the summary if no heading matches"`
}

// validate rejects unknown values instead of silently applying the default
func (p fetchParams) validate() error {
	if p.Mode != "" && p.Mode != modeMarkdown && p.Mode != modeRaw {
		return fmt.Errorf("invalid mode: %s", p.Mode)
	}
	if p.Details != "" && p.Details != detailsSummary && p.Details != detailsFull {
		return fmt.Errorf("invalid details: %s", p.Details)
	}
	return nil
}

// fetchTool reads a documentation page and returns it as markdown
func fetchTool(log *util.Logger) func(context.Context, *mcp.CallToolRequest, fetchParams) (*mcp.CallToolResult, any, error) {
	client := request.NewHelper(log)

	// redirects must not leave the allowed site
	client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		return validateUrl(req.URL)
	}

	return func(ctx context.Context, _ *mcp.CallToolRequest, params fetchParams) (*mcp.CallToolResult, any, error) {
		if err := params.validate(); err != nil {
			return nil, nil, err
		}

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

		// relative links resolve against the url the response actually came from
		text, err := documentText(resp.Request.URL, body, resp.Header.Get("Content-Type"), params)
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

// documentText converts a html document to markdown and reduces it to the requested
// detail level. Raw mode and other content types are returned unchanged.
func documentText(uri *url.URL, body []byte, contentType string, params fetchParams) (string, error) {
	if params.Mode == modeRaw || !strings.Contains(contentType, "text/html") {
		return truncate(string(body)), nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	// moved pages answer with a meta refresh instead of a redirect
	if refresh, ok := doc.Find(`meta[http-equiv="refresh"]`).Attr("content"); ok {
		if _, target, found := strings.Cut(refresh, "url="); found {
			if link, err := uri.Parse(strings.TrimSpace(target)); err == nil && validateUrl(link) == nil {
				return "[moved to " + link.String() + ", fetch that url]", nil
			}
		}
	}

	// the docs run astro starlight, sl-anchor-link is its heading permalink. Images and
	// icons carry no text, page chrome repeats on every page.
	doc.Find(`script, style, noscript, svg, img, nav, header, footer, hr,
		a.sl-anchor-link, .right-sidebar-container, .pagination-links,
		[role="navigation"], [role="banner"], [role="complementary"]`).Remove()

	content := doc.Find("main, article").First()
	if content.Length() == 0 {
		content = doc.Find("body")
	}
	if content.Length() == 0 {
		return "", fmt.Errorf("no content: %s", uri)
	}

	normalizeCode(content)

	// the site navigation repeats on every page, only the content links are worth listing
	links := documentLinks(uri, content)

	md, err := htmltomarkdown.ConvertNode(content.Nodes[0])
	if err != nil {
		return "", err
	}

	res := selectMarkdown(string(md), params)
	if len(links) > 0 {
		res += "\n\nLinks on this page:\n" + strings.Join(links, "\n")
	}

	return res, nil
}

// selectMarkdown reduces the page to the requested section or heading outline
func selectMarkdown(md string, params fetchParams) string {
	md = strings.TrimSpace(md)

	if params.Query != "" {
		if section, ok := markdownSection(md, params.Query); ok {
			return truncate(section)
		}
		return markdownSummary(md, fmt.Sprintf("no heading matches %q", params.Query))
	}

	if params.Details == detailsFull {
		return truncate(md)
	}

	return markdownSummary(md, "")
}

// markdownSummary lists the headings of the page and states what reading it in full costs
func markdownSummary(md, note string) string {
	lines, levels := splitMarkdown(md)

	var count int
	var sb strings.Builder

	for i, level := range levels {
		if level > 0 && level <= maxSummaryLevel {
			sb.WriteString(strings.TrimRight(lines[i], " ") + "\n")
			count++
		}
	}

	if note != "" {
		note += ", "
	}

	// len/4 is the usual token estimate, good enough to budget the follow-up call
	sb.WriteString(fmt.Sprintf("\n[%ssummary of %d headings, reading the entire page costs ~%d tokens. "+
		"Use details=full for the entire page or query=<heading> for a single section]",
		note, count, len(md)/4))

	return sb.String()
}

// markdownSection returns the first heading matching query with all of its subsections
func markdownSection(md, query string) (string, bool) {
	lines, levels := splitMarkdown(md)
	query = strings.ToLower(strings.TrimSpace(query))

	for i, level := range levels {
		if level == 0 || !strings.Contains(strings.ToLower(headingText(lines[i])), query) {
			continue
		}

		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if levels[j] > 0 && levels[j] <= level {
				end = j
				break
			}
		}

		return strings.TrimSpace(strings.Join(lines[i:end], "\n")), true
	}

	return "", false
}

// splitMarkdown returns the lines of the document and their heading level, zero for
// non-headings. Fenced code is skipped, yaml comments are not headings.
func splitMarkdown(md string) ([]string, []int) {
	lines := strings.Split(md, "\n")
	levels := make([]int, len(lines))

	var fence string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case fence != "":
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```"), strings.HasPrefix(trimmed, "~~~"):
			fence = trimmed[:3]
		default:
			if level := len(trimmed) - len(strings.TrimLeft(trimmed, "#")); level > 0 && level <= 6 &&
				strings.HasPrefix(trimmed[level:], " ") {
				levels[i] = level
			}
		}
	}

	return lines, levels
}

// headingText strips the leading hashes of a markdown heading
func headingText(line string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
}

// normalizeCode reduces a code block to its plain text so that the conversion emits a
// single fenced block. Highlighters wrap every line, those need an explicit break.
func normalizeCode(content *goquery.Selection) {
	content.Find("pre").Each(func(_ int, pre *goquery.Selection) {
		pre.Find("div").Not("div div").AfterHtml("\n")

		lang, ok := pre.Attr("data-language")
		if !ok {
			class, _ := pre.Find("code").Attr("class")
			_, lang, _ = strings.Cut(class, "language-")
		}

		var attr string
		if lang != "" {
			attr = ` class="language-` + html.EscapeString(lang) + `"`
		}

		pre.SetHtml("<code" + attr + ">" + html.EscapeString(strings.TrimSpace(pre.Text())) + "</code>")
	})
}

// documentLinks makes the links of the document absolute and returns the unique ones.
// Links leaving the site lose their target so that only their text is converted.
func documentLinks(uri *url.URL, content *goquery.Selection) []string {
	var res []string
	seen := make(map[string]bool)

	content.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")

		link, err := uri.Parse(href)
		if err != nil || validateUrl(link) != nil {
			a.ReplaceWithSelection(a.Contents())
			return
		}

		a.SetAttr("href", link.String())

		link.Fragment = ""
		if s := link.String(); !seen[s] && len(res) < maxFetchLinks {
			seen[s] = true
			res = append(res, s)
		}
	})

	return res
}

func truncate(s string) string {
	if len(s) <= maxFetchText {
		return s
	}
	return s[:maxFetchText] + "\n[truncated, use query=<heading> or fetch a more specific page for the full text]"
}
