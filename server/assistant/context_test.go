package assistant

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestContextChars(t *testing.T) {
	msgs := []*schema.Message{
		schema.SystemMessage("1234567890"),
		schema.AssistantMessage("", []schema.ToolCall{
			{Function: schema.FunctionCall{Name: "ab", Arguments: "cde"}},
		}),
		schema.ToolMessage("1234", "id", schema.WithToolName("ab")),
	}
	assert.Equal(t, 10+5+4+2, contextChars(msgs, nil))

	tools := []tool{{Name: "x", Description: "yz", Parameters: map[string]any{}}}
	// name + description + "{}"
	assert.Equal(t, 3+2, contextChars(nil, tools))
}
