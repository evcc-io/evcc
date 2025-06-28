package mcp

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func NewJsonResourceContents(uri string, data any) (mcp.TextResourceContents, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return mcp.TextResourceContents{}, err
	}

	return mcp.TextResourceContents{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(b),
	}, nil
}
