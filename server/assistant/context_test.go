package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestContextChars(t *testing.T) {
	msgs := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "1234567890"),
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
			llms.ToolCall{FunctionCall: &llms.FunctionCall{Name: "ab", Arguments: "cde"}},
		}},
		{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{
			llms.ToolCallResponse{Name: "ab", Content: "1234"},
		}},
	}
	assert.Equal(t, 10+5+6, contextChars(msgs, nil))

	tools := []llms.Tool{{Function: &llms.FunctionDefinition{Name: "x", Description: "yz", Parameters: map[string]any{}}}}
	// name + description + "{}"
	assert.Equal(t, 3+2, contextChars(nil, tools))
}
