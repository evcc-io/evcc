package mcp

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed prompt.tpl
var prompt string

func promptHandler() mcpsdk.PromptHandler {
	tmpl := template.Must(template.New("out").Funcs(sprig.FuncMap()).Parse(prompt))

	return func(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.GetPromptParams) (*mcpsdk.GetPromptResult, error) {
		// tmpl, err := template.New("out").Funcs(sprig.FuncMap()).Parse(prompt)
		// if err != nil {
		// 	return nil, fmt.Errorf("error parsing template: %v", err)
		// }

		out := new(bytes.Buffer)
		if err := tmpl.Execute(out, params.Arguments); err != nil {
			return nil, fmt.Errorf("failed executing template: %v", err)
		}

		return &mcpsdk.GetPromptResult{
			Messages: []*mcpsdk.PromptMessage{
				{
					Role:    "assistant",
					Content: &mcpsdk.TextContent{Text: out.String()},
				},
			},
		}, nil
	}
}
