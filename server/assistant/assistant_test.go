package assistant

import (
	"context"
	"fmt"
	"strings"
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

	// the model only sees the meta tools, the mcp tools are searchable
	assert.ElementsMatch(t, []string{findToolsName, callToolName}, toolNames(a.tools))
	assert.ElementsMatch(t, []string{"getSoc", "failing"}, toolNames(a.catalog))

	res, err := a.Chat(ctx, []Message{{Role: "user", Content: "what is the soc?"}})
	require.NoError(t, err)
	assert.Equal(t, "The vehicle is at 42%.", res.Content)

	// the tool round is reported as intermediate work
	require.Len(t, res.Steps, 1)
	assert.Equal(t, []Call{{Name: "getSoc", Arguments: `{"loadpoint":1}`}}, res.Steps[0].Calls)

	// second round trip contains the tool result
	second := llm.seen[1]
	require.Len(t, second, 4) // system, user, assistant tool call, tool response

	response, ok := second[3].Parts[0].(llms.ToolCallResponse)
	require.True(t, ok)
	assert.Equal(t, "soc of loadpoint 1 is 42", response.Content)
}

// repeat builds n identical tool calling responses
func repeat(n int, content string) []*llms.ContentResponse {
	call := llms.ToolCall{
		ID:           "1",
		Type:         "function",
		FunctionCall: &llms.FunctionCall{Name: "getSoc", Arguments: `{"loadpoint":1}`},
	}

	res := make([]*llms.ContentResponse, 0, n)
	for range n {
		res = append(res, &llms.ContentResponse{
			Choices: []*llms.ContentChoice{{Content: content, ToolCalls: []llms.ToolCall{call}}},
		})
	}

	return res
}

func TestRepeatedCallsStop(t *testing.T) {
	llm := &fakeLLM{responses: repeat(maxIterations, "still looking")}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	res, err := a.Chat(t.Context(), []Message{{Role: "user", Content: "soc?"}})
	require.NoError(t, err)

	// two identical rounds end the loop, a third call without tools produces the answer
	assert.Equal(t, "still looking", res.Content)
	assert.Len(t, llm.seen, 3)
	last := llm.seen[2][len(llm.seen[2])-1]
	assert.Equal(t, llms.ChatMessageTypeHuman, last.Role, "the last call carries the nudge")
	assert.Len(t, res.Steps, 2)
}

func TestRepeatedCallsWithoutContent(t *testing.T) {
	llm := &fakeLLM{responses: repeat(maxIterations, "")}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	// nothing to answer with, the caller has to see the failure
	_, err = a.Chat(t.Context(), []Message{{Role: "user", Content: "soc?"}})
	assert.EqualError(t, err, "repeated tool calls")
}

func TestExhaustedRoundsAnswer(t *testing.T) {
	// distinct calls every round, so the loop runs out instead of detecting a stall
	res := make([]*llms.ContentResponse, 0, maxIterations+1)
	for i := range maxIterations {
		res = append(res, &llms.ContentResponse{Choices: []*llms.ContentChoice{{
			Content: fmt.Sprintf("checking %d", i),
			ToolCalls: []llms.ToolCall{{
				ID:           "1",
				Type:         "function",
				FunctionCall: &llms.FunctionCall{Name: "getSoc", Arguments: fmt.Sprintf(`{"loadpoint":%d}`, i)},
			}},
		}}})
	}

	// the call without tools concludes from what was gathered
	res = append(res, &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{Content: "the soc is 42%"}},
	})

	llm := &fakeLLM{responses: res}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	out, err := a.Chat(t.Context(), []Message{{Role: "user", Content: "soc?"}})
	require.NoError(t, err)

	// the rounds are not discarded, they are concluded
	assert.Equal(t, "the soc is 42%", out.Content)
	assert.Len(t, llm.seen, maxIterations+1)
	assert.Len(t, out.Steps, maxIterations)
}

func TestEmptyAnswerRetries(t *testing.T) {
	llm := &fakeLLM{responses: []*llms.ContentResponse{
		// everything in the reasoning, nothing said out loud
		{Choices: []*llms.ContentChoice{{Content: "\n\n", ReasoningContent: "the mode is pv"}}},
		{Choices: []*llms.ContentChoice{{Content: "The mode is pv."}}},
	}}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	res, err := a.Chat(t.Context(), []Message{{Role: "user", Content: "mode?"}})
	require.NoError(t, err)

	assert.Equal(t, "The mode is pv.", res.Content)
	assert.Len(t, llm.seen, 2)
	assert.Equal(t, "the mode is pv", res.Steps[0].Reasoning)
}

func TestSilentModelFails(t *testing.T) {
	llm := &fakeLLM{responses: []*llms.ContentResponse{
		{Choices: []*llms.ContentChoice{{Content: "  "}}},
		{Choices: []*llms.ContentChoice{{Content: ""}}},
	}}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	_, err = a.Chat(t.Context(), []Message{{Role: "user", Content: "mode?"}})
	assert.EqualError(t, err, "empty answer")
}

func TestTruncateResult(t *testing.T) {
	assert.Equal(t, "short", truncateResult("short"))

	res := truncateResult(strings.Repeat("x", maxToolResult+1))
	assert.True(t, strings.HasPrefix(res, strings.Repeat("x", maxToolResult)))
	assert.Contains(t, res, "[truncated")
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

func TestOllamaUrl(t *testing.T) {
	for base, want := range map[string]string{
		"":                         "http://localhost:11434/v1",
		"http://nas:11434":         "http://nas:11434/v1",
		"http://nas:11434/":        "http://nas:11434/v1",
		"http://nas:11434/v1":      "http://nas:11434/v1",
		"https://ollama.local/v1/": "https://ollama.local/v1",
	} {
		assert.Equal(t, want, ollamaUrl(base), base)
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

func TestCustomWithoutToken(t *testing.T) {
	// local OpenAI-compatible endpoints need no key, langchaingo rejects an empty token
	_, err := newLLM(Config{Provider: Custom, Model: "m", BaseUrl: "http://localhost:1234/v1"})
	require.NoError(t, err)
}

func TestRedacted(t *testing.T) {
	cfg := Config{Provider: OpenAI, Model: "gpt", Token: "secret", BaseUrl: "http://x"}
	assert.NotContains(t, cfg.Redacted().(Config).Token, "secret")
}

// call invokes a tool the way the model would
func call(t *testing.T, a *Assistant, name, args string) string {
	t.Helper()
	return a.callTool(t.Context(), llms.ToolCall{
		FunctionCall: &llms.FunctionCall{Name: name, Arguments: args},
	})
}

func TestFindTools(t *testing.T) {
	a, err := newAssistant(t.Context(), &fakeLLM{}, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	// name and description are searchable
	assert.Contains(t, call(t, a, findToolsName, `{"query":"soc"}`), "getSoc")
	assert.Contains(t, call(t, a, findToolsName, `{"query":"vehicle soc"}`), "vehicle soc")

	// the schema comes with the result, callTool cannot be built without it
	assert.Contains(t, call(t, a, findToolsName, `{"query":"soc"}`), `"loadpoint"`)

	// the model gets told when nothing matches instead of an empty result
	assert.Contains(t, call(t, a, findToolsName, `{"query":"nonsense"}`), "no tool matches")
}

func TestCallSignatures(t *testing.T) {
	calls := []llms.ToolCall{
		{FunctionCall: &llms.FunctionCall{Name: "getSoc", Arguments: `{"loadpoint":1}`}},
		{FunctionCall: nil},
	}

	// a call without a function is dropped, it cannot be executed either
	assert.Equal(t, []Call{{Name: "getSoc", Arguments: `{"loadpoint":1}`}}, roundCalls(calls))
	assert.Equal(t, []string{`getSoc({"loadpoint":1})`}, callSignatures(roundCalls(calls)))
	assert.Equal(t, `{"loadpoint":1}`, logArgs(map[string]any{"loadpoint": 1}))
}

func TestCallToolDispatch(t *testing.T) {
	a, err := newAssistant(t.Context(), &fakeLLM{}, testServer(t))
	require.NoError(t, err)
	defer a.Close()

	const want = "soc of loadpoint 1 is 42"

	// arguments as object and, for weaker models, as json string
	assert.Equal(t, want, call(t, a, callToolName, `{"name":"getSoc","arguments":{"loadpoint":1}}`))
	assert.Equal(t, want, call(t, a, callToolName, `{"name":"getSoc","arguments":"{\"loadpoint\":1}"}`))

	// tools found in the catalog remain callable directly
	assert.Equal(t, want, call(t, a, "getSoc", `{"loadpoint":1}`))

	// failure modes stay text for the model
	assert.Contains(t, call(t, a, callToolName, `{"arguments":{}}`), "missing tool name")
	assert.Contains(t, call(t, a, callToolName, `{"name":"nope"}`), "error:")
	assert.Contains(t, call(t, a, callToolName, `{"name":"getSoc","arguments":"{"}`), "invalid arguments")
}
