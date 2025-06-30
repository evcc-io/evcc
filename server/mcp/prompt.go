package mcp

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed prompt.tpl
var prompt string

func promptHandler() server.PromptHandlerFunc {
	tmpl := template.Must(template.New("out").Funcs(sprig.FuncMap()).Parse(prompt))

	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// tmpl, err := template.New("out").Funcs(sprig.FuncMap()).Parse(prompt)
		// if err != nil {
		// 	return nil, fmt.Errorf("error parsing template: %v", err)
		// }

		out := new(bytes.Buffer)
		if err := tmpl.Execute(out, req.Params.Arguments); err != nil {
			return nil, fmt.Errorf("failed executing template: %v", err)
		}

		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: out.String(),
					},
				},
			},
		}, nil
	}
}
