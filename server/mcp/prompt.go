package mcp

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed prompt.tpl
var prompt string

func promptHandler() mcp.PromptHandler {
	tmpl := template.Must(template.New("out").Funcs(sprig.FuncMap()).Parse(prompt))

	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParams) (*mcp.GetPromptResult, error) {
		out := new(bytes.Buffer)
		if err := tmpl.Execute(out, params.Arguments); err != nil {
			return &mcp.GetPromptResult{
				Messages: []*mcp.PromptMessage{
					{
						Role: "assistant",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf("The was an error parsing the request: %v", err),
						},
					},
				},
			}, nil
		}

		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{
					Role:    "assistant",
					Content: &mcp.TextContent{Text: out.String()},
				},
			},
		}, nil
	}
}
