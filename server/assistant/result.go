package assistant

import (
	"encoding/json"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxElements is the number of array elements kept per array in a tool result.
// A full state response is mostly arrays and blows the context of local models,
// the remainder is fetched with a jq filter once the model knows what it wants.
const maxElements = 3

// shorten cuts long arrays out of a json tool result. Anything that is not json
// is returned unchanged, as is a result that has no array worth cutting.
func shorten(s string, jq bool) string {
	var doc any
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return s
	}

	doc, dropped := shortenValue(doc)
	if dropped == 0 {
		return s
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return s
	}

	var sb strings.Builder
	sb.Write(b)
	fmt.Fprintf(&sb, "\n\n[%d array elements omitted, %d shown per array", dropped, maxElements)
	if jq {
		sb.WriteString(`. Call the tool again with a jq filter for the full array, e.g. {"jq": ".loadpoints"}`)
	}
	sb.WriteString("]")

	return sb.String()
}

// structuredBody returns the payload openapi tools report next to their text
// content. The text wraps the payload in a request envelope the model does not
// need and cannot be parsed back reliably.
func structuredBody(res *mcpsdk.CallToolResult) (string, bool) {
	structured, ok := res.StructuredContent.(map[string]any)
	if !ok {
		return "", false
	}

	body, ok := structured["body"]
	if !ok {
		return "", false
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", false
	}

	return string(b), true
}

// hintEmpty tells the model how to recover from a filter that selected nothing.
// Without it a guessed jq path is a dead end, the tools describe no structure.
func hintEmpty(s string, args map[string]any) string {
	if _, ok := args["jq"]; !ok {
		return s
	}

	lines := strings.Split(strings.TrimSpace(s), "\n")

	switch strings.TrimSpace(lines[len(lines)-1]) {
	case "null", "[]", "{}", "":
		return s + "\n\n[the jq filter selected nothing. Call the tool without a filter to see which fields exist]"
	default:
		return s
	}
}

// shortenValue keeps the first maxElements of every array and reports how many
// elements were dropped. Truncated arrays end in a marker naming the remainder.
func shortenValue(v any) (any, int) {
	var dropped int

	switch t := v.(type) {
	case []any:
		for i, e := range t {
			e, d := shortenValue(e)
			t[i], dropped = e, dropped+d
		}

		if rest := len(t) - maxElements; rest > 0 {
			dropped += rest
			t = append(t[:maxElements:maxElements], fmt.Sprintf("… %d more elements", rest))
		}

		return t, dropped

	case map[string]any:
		for k, e := range t {
			e, d := shortenValue(e)
			t[k], dropped = e, dropped+d
		}

		return t, dropped
	}

	return v, 0
}

// supportsJq reports whether the tool accepts a jq filter to select what it returns
func (a *Assistant) supportsJq(name string) bool {
	for _, tool := range a.catalog {
		if tool.Name != name {
			continue
		}
		return hasProperty(tool, "jq")
	}

	return false
}

func hasProperty(t tool, name string) bool {
	props, ok := t.Parameters["properties"].(map[string]any)
	if !ok {
		return false
	}

	_, ok = props[name]

	return ok
}
