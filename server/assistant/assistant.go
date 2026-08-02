package assistant

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/langchaingo/llms"
)

// maxIterations limits the tool calling loop
const maxIterations = 12

// maxHistory bounds the characters of past messages resent with each question,
// roughly a quarter of that in tokens. The tool definitions sit at the start of
// the prompt, so a conversation that outgrows the model's context evicts them
// first and the model silently loses its tools.
const maxHistory = 4000

// systemPrompt is embedded at build time, edit prompt.txt to change it
//
//go:embed prompt.txt
var systemPrompt string

// Message is a single chat message
type Message struct {
	Role    string `json:"role"` // user|assistant
	Content string `json:"content"`
}

// Call is a tool the model asked for
type Call struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// Step is the intermediate work of a single round, shown alongside the answer
type Step struct {
	Reasoning string `json:"reasoning,omitempty"`
	Calls     []Call `json:"calls,omitempty"`
}

// Result is the answer and how the model arrived at it
type Result struct {
	Content string `json:"content"`
	Steps   []Step `json:"steps,omitempty"`
}

// Assistant answers questions about the running system using evcc's MCP tools
type Assistant struct {
	log     *util.Logger
	mcpLog  *util.Logger
	llm     llms.Model
	client  *mcpsdk.ClientSession
	server  *mcpsdk.ServerSession
	tools   []llms.Tool // offered to the model
	catalog []llms.Tool // searchable through findTools
	context string
}

// New connects a language model to the given MCP server
func New(ctx context.Context, cfg Config, srv *mcpsdk.Server) (*Assistant, error) {
	llm, err := newLLM(cfg)
	if err != nil {
		return nil, err
	}
	return newAssistant(ctx, llm, srv)
}

func newAssistant(ctx context.Context, llm llms.Model, srv *mcpsdk.Server) (*Assistant, error) {
	ct, st := mcpsdk.NewInMemoryTransports()

	// server must be connected before the client initializes the session
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		return nil, err
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "evcc-assistant", Version: util.Version}, nil)

	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		ss.Close()
		return nil, err
	}

	catalog, err := listTools(ctx, cs)
	if err != nil {
		cs.Close()
		ss.Close()
		return nil, err
	}

	return &Assistant{
		log: util.NewLogger("assistant"),
		// tool calls are mcp requests, log them with the mcp transport they trigger
		mcpLog:  util.NewLogger("mcp"),
		llm:     llm,
		client:  cs,
		server:  ss,
		tools:   offeredTools(catalog),
		catalog: catalog,
	}, nil
}

// WithContext adds situational context, e.g. the ui page the question was asked from
func (a *Assistant) WithContext(s string) *Assistant {
	a.context = s
	return a
}

// Close releases the MCP session
func (a *Assistant) Close() {
	a.client.Close()
	a.server.Close()
}

func listTools(ctx context.Context, cs *mcpsdk.ClientSession) ([]llms.Tool, error) {
	var res []llms.Tool

	for params := new(mcpsdk.ListToolsParams); ; {
		list, err := cs.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, t := range list.Tools {
			schema, ok := t.InputSchema.(map[string]any)
			if !ok {
				schema = map[string]any{"type": "object"}
			}

			res = append(res, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  schema,
				},
			})
		}

		if list.NextCursor == "" {
			return res, nil
		}
		params = &mcpsdk.ListToolsParams{Cursor: list.NextCursor}
	}
}

// trimHistory drops the oldest messages that do not fit the budget. The current
// question is always kept, even when it exceeds the budget on its own.
func trimHistory(history []Message, budget int) []Message {
	if len(history) < 2 {
		return history
	}

	// the question is sent no matter its size, older messages have to fit
	budget -= len(history[len(history)-1].Content)

	for i := len(history) - 2; i >= 0; i-- {
		if budget -= len(history[i].Content); budget < 0 {
			return history[i+1:]
		}
	}

	return history
}

// roundCalls describes the tools a round wants to call. Meta tool calls carry their
// target in the arguments, it is resolved in callTool.
func roundCalls(calls []llms.ToolCall) []Call {
	res := make([]Call, 0, len(calls))

	for _, tc := range calls {
		if tc.FunctionCall != nil {
			res = append(res, Call{
				Name:      tc.FunctionCall.Name,
				Arguments: strings.TrimSpace(tc.FunctionCall.Arguments),
			})
		}
	}

	return res
}

// callSignatures renders the calls of a round for the log
func callSignatures(calls []Call) []string {
	res := make([]string, 0, len(calls))

	for _, c := range calls {
		res = append(res, c.Name+"("+c.Arguments+")")
	}

	return res
}

// toolNames lists the names of the given tools
func toolNames(tools []llms.Tool) []string {
	res := make([]string, 0, len(tools))

	for _, tool := range tools {
		res = append(res, tool.Function.Name)
	}

	return res
}

// logArgs renders tool arguments as json, the map order of %v is not stable
func logArgs(args map[string]any) string {
	b, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}

	return string(b)
}

// Chat answers the conversation, calling tools as needed
func (a *Assistant) Chat(ctx context.Context, history []Message) (Result, error) {
	prompt := strings.TrimSpace(systemPrompt)
	if a.context != "" {
		prompt += "\n\nContext for this conversation:\n" + a.context
	}

	messages := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, prompt)}

	if trimmed := trimHistory(history, maxHistory); len(trimmed) < len(history) {
		a.log.DEBUG.Printf("history trimmed to the last %d of %d messages", len(trimmed), len(history))
		history = trimmed
	}

	for _, m := range history {
		role := llms.ChatMessageTypeHuman
		if m.Role == "assistant" {
			role = llms.ChatMessageTypeAI
		}
		messages = append(messages, llms.TextParts(role, m.Content))
	}

	a.log.DEBUG.Printf("chat: %d messages, %d tools offered", len(history), len(a.tools))
	a.log.TRACE.Printf("tools offered: %v", toolNames(a.tools))
	a.log.TRACE.Printf("prompt: %s", prompt)

	var steps []Step

	// keeps a step out of the result when the model reports neither reasoning nor calls
	addStep := func(step Step) {
		if step.Reasoning != "" || len(step.Calls) > 0 {
			steps = append(steps, step)
		}
	}

	for round := range maxIterations {
		resp, err := a.llm.GenerateContent(ctx, messages, llms.WithTools(a.tools))
		if err != nil {
			a.log.ERROR.Println(err)
			return Result{}, err
		}
		if len(resp.Choices) == 0 {
			return Result{}, errors.New("empty response")
		}

		choice := resp.Choices[0]
		reasoning := strings.TrimSpace(choice.ReasoningContent)

		if len(choice.ToolCalls) == 0 {
			// zero rounds means the model answered from its own knowledge
			a.log.DEBUG.Printf("answer after %d tool rounds", round)
			addStep(Step{Reasoning: reasoning})

			return Result{Content: choice.Content, Steps: steps}, nil
		}

		calls := roundCalls(choice.ToolCalls)
		a.log.DEBUG.Printf("tool round %d: %v", round+1, callSignatures(calls))
		addStep(Step{Reasoning: reasoning, Calls: calls})

		msg := llms.TextParts(llms.ChatMessageTypeAI, choice.Content)
		for _, tc := range choice.ToolCalls {
			msg.Parts = append(msg.Parts, tc)
		}
		messages = append(messages, msg)

		for _, tc := range choice.ToolCalls {
			messages = append(messages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       tc.FunctionCall.Name,
					Content:    a.callTool(ctx, tc),
				}},
			})
		}
	}

	err := fmt.Errorf("exceeded %d tool call rounds", maxIterations)
	a.log.ERROR.Println(err)

	return Result{}, err
}

// callTool executes a tool call and returns its textual result. Errors are returned to the model, not to the caller.
func (a *Assistant) callTool(ctx context.Context, tc llms.ToolCall) string {
	if tc.FunctionCall == nil {
		a.mcpLog.ERROR.Println("missing function call")
		return "error: missing function call"
	}

	name := tc.FunctionCall.Name

	var args map[string]any
	if s := strings.TrimSpace(tc.FunctionCall.Arguments); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			a.mcpLog.ERROR.Printf("tool %s invalid arguments: %v", name, err)
			return "error: invalid arguments: " + err.Error()
		}
	}

	// the model only sees the meta tools, everything else is reached through them
	switch name {
	case findToolsName:
		a.mcpLog.DEBUG.Printf("%s %s", name, logArgs(args))
		return a.findTools(args)

	case callToolName:
		var err error
		if name, args, err = dispatchArgs(args); err != nil {
			a.mcpLog.ERROR.Printf("%s: %v", callToolName, err)
			return "error: " + err.Error()
		}
	}

	a.mcpLog.DEBUG.Printf("tool %s %s", name, logArgs(args))

	res, err := a.client.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		a.mcpLog.ERROR.Printf("tool %s: %v", name, err)
		return "error: " + err.Error()
	}

	var sb strings.Builder
	for _, c := range res.Content {
		if txt, ok := c.(*mcpsdk.TextContent); ok {
			sb.WriteString(txt.Text)
		}
	}

	if res.IsError {
		a.mcpLog.ERROR.Printf("tool %s: %s", name, sb.String())
		return "error: " + sb.String()
	}

	// openapi tools report the payload separately, the text content wraps it in an
	// envelope that only costs the model context
	result := sb.String()
	if body, ok := structuredBody(res); ok {
		result = body
	}

	a.mcpLog.TRACE.Printf("tool %s result: %s", name, result)

	// the model gets the result with long arrays cut down, the log keeps the original
	return hintEmpty(shorten(result, a.supportsJq(name)), args)
}
