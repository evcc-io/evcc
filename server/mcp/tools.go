package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func docsTool(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewResourceLink("https://docs.evcc.io", "evcc-docs", "evcc documentation", "text/html"),
		},
	}, nil
}

func samplingTool(srv *server.MCPServer) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		question, err := request.RequireString("question")
		if err != nil {
			return nil, err
		}

		systemPrompt := request.GetString("system_prompt", "You are a home energy management system. Use available tools and resources to help the user planning energy consumption, optimizing costs and managing battery usage.")

		// Create sampling request
		samplingRequest := mcp.CreateMessageRequest{
			CreateMessageParams: mcp.CreateMessageParams{
				Messages: []mcp.SamplingMessage{
					{
						Role: mcp.RoleUser,
						Content: mcp.TextContent{
							Type: "text",
							Text: question,
						},
					},
				},
				SystemPrompt: systemPrompt,
				MaxTokens:    1000,
			},
		}

		// Request sampling from client
		result, err := srv.RequestSampling(ctx, samplingRequest)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error requesting sampling: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Return the LLM response
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("LLM Response: %s", getTextFromContent(result)),
				},
			},
		}, nil
	}
}

func getTextFromContent(content any) string {
	return fmt.Sprintf("%T %v", content, content)
	// var text string
	// for _, c := range content {
	// 	if t, ok := c.(mcp.TextContent); ok {
	// 		text += t.Text + "\n"
	// 	}
	// }
	// return text
}
