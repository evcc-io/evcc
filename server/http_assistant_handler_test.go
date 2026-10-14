package server

import (
	"testing"

	"github.com/evcc-io/evcc/server/assistant"
	"github.com/stretchr/testify/assert"
)

func TestAssistantToken(t *testing.T) {
	stored := assistant.Config{Provider: assistant.OpenAI, BaseUrl: "", Token: "secret"}

	for _, tc := range []struct {
		req  assistant.Config
		want string
	}{
		// unchanged token, same endpoint
		{assistant.Config{Provider: assistant.OpenAI, Token: masked}, "secret"},

		// a url named in the request must not receive the stored token
		{assistant.Config{Provider: assistant.OpenAI, BaseUrl: "https://evil.example/v1", Token: masked}, ""},

		// nor may another provider claim it
		{assistant.Config{Provider: assistant.Anthropic, Token: masked}, ""},

		// a token supplied by the caller is used as is
		{assistant.Config{Provider: assistant.OpenAI, BaseUrl: "https://evil.example/v1", Token: "own"}, "own"},
		{assistant.Config{Provider: assistant.Ollama}, ""},
	} {
		assert.Equal(t, tc.want, assistantToken(tc.req, stored), tc.req)
	}
}
