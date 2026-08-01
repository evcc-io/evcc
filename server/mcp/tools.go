package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// docsHint tells the model how to actually read the documentation. Clients that only
// evaluate text content see nothing but the resource link otherwise.
const docsHint = `The evcc documentation is at https://docs.evcc.io.
Use the fetchDocs tool to read a page, start at https://docs.evcc.io/en for the table of contents.
https://docs.evcc.io/llms.txt lists the machine-readable documentation sets.`

func docsTool(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: docsHint},
			&mcp.ResourceLink{
				URI:      "https://docs.evcc.io",
				Name:     "evcc-docs",
				Title:    "evcc documentation",
				MIMEType: "text/html",
			},
		},
	}, nil, nil
}
