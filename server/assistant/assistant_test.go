package assistant

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

type socInput struct {
	Loadpoint int `json:"loadpoint" jsonschema:"loadpoint id"`
}

// fakeLLM replays canned responses and records the messages it was given
type fakeLLM struct {
	responses []*llms.ContentResponse
	seen      [][]llms.MessageContent
}

func (f *fakeLLM) GenerateContent(_ context.Context, messages []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	f.seen = append(f.seen, messages)
	res := f.responses[0]
	f.responses = f.responses[1:]
	return res, nil
}

func (f *fakeLLM) Call(context.Context, string, ...llms.CallOption) (string, error) {
	panic("not implemented")
}

func testServer(t *testing.T) *mcpsdk.Server {
	t.Helper()

	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{Name: "getSoc", Description: "vehicle soc"},
		func(_ context.Context, _ *mcpsdk.CallToolRequest, in socInput) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "soc of loadpoint 1 is 42"}},
			}, nil, nil
		})

	mcpsdk.AddTool(srv, &mcpsdk.Tool{Name: "failing", Description: "always fails"},
		func(_ context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "meter unavailable"}},
			}, nil, nil
		})

	return srv
}

func TestToolLoop(t *testing.T) {
	ctx := t.Context()

	toolCall := llms.ToolCall{
		ID:           "1",
		Type:         "function",
		FunctionCall: &llms.FunctionCall{Name: "getSoc", Arguments: `{"loadpoint":1}`},
	}

	llm := &fakeLLM{responses: []*llms.ContentResponse{
		{Choices: []*llms.ContentChoice{{ToolCalls: []llms.ToolCall{toolCall}}}},
		{Choices: []*llms.ContentChoice{{Content: "The vehicle is at 42%."}}},
	}}

	a, err := newAssistant(ctx, llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	// mcp tools are exposed to the model
	names := make([]string, 0, len(a.tools))
	for _, t := range a.tools {
		names = append(names, t.Function.Name)
	}
	assert.ElementsMatch(t, []string{"getSoc", "failing"}, names)

	res, err := a.Chat(ctx, []Message{{Role: "user", Content: "what is the soc?"}})
	require.NoError(t, err)
	assert.Equal(t, "The vehicle is at 42%.", res)

	// second round trip contains the tool result
	second := llm.seen[1]
	require.Len(t, second, 4) // system, user, assistant tool call, tool response

	response, ok := second[3].Parts[0].(llms.ToolCallResponse)
	require.True(t, ok)
	assert.Equal(t, "soc of loadpoint 1 is 42", response.Content)
}

func TestToolError(t *testing.T) {
	ctx := t.Context()

	a, err := newAssistant(ctx, &fakeLLM{}, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	// all failure modes are surfaced to the model as text, never as an error
	for _, tc := range []struct {
		name, args, contains string
	}{
		{"unknown", "{}", "error:"},
		{"getSoc", "{", "error: invalid arguments"},
		{"failing", "{}", "error: meter unavailable"},
	} {
		res := a.callTool(ctx, llms.ToolCall{
			FunctionCall: &llms.FunctionCall{Name: tc.name, Arguments: tc.args},
		})
		assert.Contains(t, res, tc.contains)
	}
}

func TestConfigValidate(t *testing.T) {
	for _, tc := range []struct {
		cfg Config
		ok  bool
	}{
		{Config{Provider: OpenAI, Model: "gpt", Token: "t"}, true},
		{Config{Provider: OpenAI, Model: "gpt"}, false},
		{Config{Provider: Ollama, Model: "qwen3"}, true},
		{Config{Provider: Custom, Model: "m"}, false},
		{Config{Provider: Custom, Model: "m", BaseUrl: "http://x"}, true},
		{Config{Provider: "bogus", Model: "m", Token: "t"}, false},
		{Config{Provider: Anthropic, Token: "t"}, false},
	} {
		err := tc.cfg.validate()
		if tc.ok {
			assert.NoError(t, err, tc.cfg)
		} else {
			assert.Error(t, err, tc.cfg)
		}
	}
}

func TestRedacted(t *testing.T) {
	cfg := Config{Provider: OpenAI, Model: "gpt", Token: "secret", BaseUrl: "http://x"}
	assert.NotContains(t, cfg.Redacted().(Config).Token, "secret")
}
