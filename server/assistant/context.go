package assistant

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
)

// charsPerToken is the ratio the history budget is built on, close enough to
// watch a conversation grow
const charsPerToken = 4

// contextChars is the size of everything sent with a request. The tool
// definitions count, they outgrow the messages early in a conversation
func contextChars(messages []*schema.Message, tools []tool) int {
	var res int

	for _, m := range messages {
		res += len(m.Content) + len(m.ReasoningContent) + len(m.ToolName)

		for _, tc := range m.ToolCalls {
			res += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}

	for _, t := range tools {
		res += len(t.Name) + len(t.Description)
		if b, err := json.Marshal(t.Parameters); err == nil {
			res += len(b)
		}
	}

	return res
}

// logContext reports the estimated prompt size of a request
func (a *Assistant) logContext(messages []*schema.Message, tools []tool) {
	chars := contextChars(messages, tools)
	a.log.DEBUG.Printf("context: ~%d tokens, %d messages, %d tools", chars/charsPerToken, len(messages), len(tools))
}
