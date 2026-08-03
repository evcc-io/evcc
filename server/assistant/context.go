package assistant

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

// charsPerToken is the ratio the history budget is built on, close enough to
// watch a conversation grow
const charsPerToken = 4

// contextChars is the size of everything sent with a request. The tool
// definitions count, they outgrow the messages early in a conversation
func contextChars(messages []llms.MessageContent, tools []llms.Tool) int {
	var res int

	for _, m := range messages {
		for _, part := range m.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				res += len(p.Text)
			case llms.ToolCall:
				if p.FunctionCall != nil {
					res += len(p.FunctionCall.Name) + len(p.FunctionCall.Arguments)
				}
			case llms.ToolCallResponse:
				res += len(p.Name) + len(p.Content)
			}
		}
	}

	for _, t := range tools {
		if t.Function == nil {
			continue
		}

		res += len(t.Function.Name) + len(t.Function.Description)
		if b, err := json.Marshal(t.Function.Parameters); err == nil {
			res += len(b)
		}
	}

	return res
}

// logContext reports the estimated prompt size of a request
func (a *Assistant) logContext(messages []llms.MessageContent, tools []llms.Tool) {
	chars := contextChars(messages, tools)
	a.log.DEBUG.Printf("context: ~%d tokens, %d messages, %d tools", chars/charsPerToken, len(messages), len(tools))
}
