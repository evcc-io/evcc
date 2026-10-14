package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// an unconfigured assistant answers every question the same way, whatever the
// body says. The check precedes parsing, so no request reaches the model
func TestChatHandlerUnconfigured(t *testing.T) {
	for _, body := range []string{
		`{"messages":[{"role":"user","content":"hi"}]}`,
		`{"messages":[]}`,
		`{`,
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(body))

		ChatHandler(http.NotFoundHandler())(w, r)

		assert.Equal(t, http.StatusPreconditionFailed, w.Code, body)
		assert.Contains(t, w.Body.String(), "error", body)
	}
}

// chat streams its steps as they happen and closes with the result
func TestChatHandlerStream(t *testing.T) {
	call := llms.ToolCall{
		ID:           "1",
		Type:         "function",
		FunctionCall: &llms.FunctionCall{Name: "getSoc", Arguments: `{"loadpoint":1}`},
	}

	llm := &fakeLLM{responses: []*llms.ContentResponse{
		{Choices: []*llms.ContentChoice{{ToolCalls: []llms.ToolCall{call}}}},
		{Choices: []*llms.ContentChoice{{Content: "The vehicle is at 42%."}}},
	}}

	a, err := newAssistant(t.Context(), llm, testServer(t))
	require.NoError(t, err)

	h := chatHandler(
		func() (Config, error) { return Config{}, nil },
		func(context.Context, Config) (*Assistant, error) { return a, nil },
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/chat",
		strings.NewReader(`{"messages":[{"role":"user","content":"soc?"}]}`))

	h(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-ndjson", w.Header().Get("Content-Type"))

	var events []chatEvent
	for _, line := range strings.Split(strings.TrimSpace(w.Body.String()), "\n") {
		var ev chatEvent
		require.NoError(t, json.Unmarshal([]byte(line), &ev), line)
		events = append(events, ev)
	}

	// the tool round arrives before the answer it produced
	require.Len(t, events, 2)
	require.NotNil(t, events[0].Step)
	assert.Equal(t, []Call{{Name: "getSoc", Arguments: `{"loadpoint":1}`}}, events[0].Step.Calls)

	require.NotNil(t, events[1].Result)
	assert.Equal(t, "The vehicle is at 42%.", events[1].Result.Content)
}
