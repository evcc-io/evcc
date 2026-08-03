package assistant

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/evcc-io/evcc/util"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxIterations limits the tool calling loop
const maxIterations = 12

// maxHistory bounds the characters of past messages resent with each question,
// roughly a quarter of that in tokens. The tool definitions sit at the start of
// the prompt, so a conversation that outgrows the model's context evicts them
// first and the model silently loses its tools.
const maxHistory = 4000

// maxToolResult bounds a single tool result handed back to the model
const maxToolResult = 4000

// maxRepeats is how often the same call may repeat before the model counts as stuck
const maxRepeats = 2

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
	llm     model.ToolCallingChatModel
	bound   model.BaseChatModel // llm with the offered tools attached
	client  *mcpsdk.ClientSession
	server  *mcpsdk.ServerSession
	tools   []tool // offered to the model
	catalog []tool // searchable through findTools
	context string
	onStep  func(Step) // reports a finished round while the answer is still being worked on
}

// New connects a language model to the given MCP server
func New(ctx context.Context, cfg Config, srv *mcpsdk.Server) (*Assistant, error) {
	llm, err := newLLM(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return newAssistant(ctx, llm, srv)
}

func newAssistant(ctx context.Context, llm model.ToolCallingChatModel, srv *mcpsdk.Server) (*Assistant, error) {
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

	// every step from here owns both sessions until the assistant does
	fail := func(err error) (*Assistant, error) {
		cs.Close()
		ss.Close()
		return nil, err
	}

	catalog, err := listTools(ctx, cs)
	if err != nil {
		return fail(err)
	}

	tools := offeredTools(catalog)

	infos, err := toolInfos(tools)
	if err != nil {
		return fail(err)
	}

	// WithTools returns a new instance, the bare one answers the concluding call
	bound, err := llm.WithTools(infos)
	if err != nil {
		return fail(err)
	}

	return &Assistant{
		log: util.NewLogger("assistant"),
		// tool calls are mcp requests, log them with the mcp transport they trigger
		mcpLog:  util.NewLogger("mcp"),
		llm:     llm,
		bound:   bound,
		client:  cs,
		server:  ss,
		tools:   tools,
		catalog: catalog,
	}, nil
}

// WithContext adds situational context, e.g. the ui page the question was asked from
func (a *Assistant) WithContext(s string) *Assistant {
	a.context = s
	return a
}

// WithSteps reports each round as it finishes, the caller can show the work before the answer
func (a *Assistant) WithSteps(f func(Step)) *Assistant {
	a.onStep = f
	return a
}

// Close releases the MCP session
func (a *Assistant) Close() {
	a.client.Close()
	a.server.Close()
}

func listTools(ctx context.Context, cs *mcpsdk.ClientSession) ([]tool, error) {
	var res []tool

	for params := new(mcpsdk.ListToolsParams); ; {
		list, err := cs.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, t := range list.Tools {
			params, ok := t.InputSchema.(map[string]any)
			if !ok {
				params = map[string]any{"type": "object"}
			}

			res = append(res, tool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
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
func roundCalls(calls []schema.ToolCall) []Call {
	res := make([]Call, 0, len(calls))

	for _, tc := range calls {
		// a nameless call cannot be executed either
		if tc.Function.Name == "" {
			continue
		}

		res = append(res, Call{
			Name:      tc.Function.Name,
			Arguments: strings.TrimSpace(tc.Function.Arguments),
		})
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
func toolNames(tools []tool) []string {
	res := make([]string, 0, len(tools))

	for _, tool := range tools {
		res = append(res, tool.Name)
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

	messages := []*schema.Message{schema.SystemMessage(prompt)}

	if trimmed := trimHistory(history, maxHistory); len(trimmed) < len(history) {
		a.log.DEBUG.Printf("history trimmed to the last %d of %d messages", len(trimmed), len(history))
		history = trimmed
	}

	for _, m := range history {
		if m.Role == "assistant" {
			messages = append(messages, schema.AssistantMessage(m.Content, nil))
		} else {
			messages = append(messages, schema.UserMessage(m.Content))
		}
	}

	a.log.DEBUG.Printf("chat: %d messages, %d tools offered", len(history), len(a.tools))
	a.log.TRACE.Printf("tools offered: %v", toolNames(a.tools))
	a.log.TRACE.Printf("prompt: %s", prompt)

	var steps []Step
	var content string
	var previous []Call
	var repeats int

	// keeps a step out of the result when the model reports neither reasoning nor calls
	addStep := func(step Step) {
		if step.Reasoning != "" || len(step.Calls) > 0 {
			steps = append(steps, step)
			if a.onStep != nil {
				a.onStep(step)
			}
		}
	}

	// the loop stopped without the model concluding. Offering no tools leaves it nothing
	// to do but answer from what it gathered, which beats reporting the last half thought.
	incomplete := func(reason string) (Result, error) {
		a.log.WARN.Printf("%s, answering without tools", reason)

		messages = append(messages, schema.UserMessage(
			"Answer the question with what you have found so far. Do not call any more tools."))

		a.logContext(messages, nil)

		if resp, err := a.llm.Generate(ctx, messages); err != nil {
			a.log.ERROR.Println(err)
		} else if resp != nil {
			if answer := strings.TrimSpace(resp.Content); answer != "" {
				return Result{Content: answer, Steps: steps}, nil
			}
		}

		if content == "" {
			return Result{}, errors.New(reason)
		}

		return Result{Content: content, Steps: steps}, nil
	}

	for round := range maxIterations {
		a.logContext(messages, a.tools)

		resp, err := a.bound.Generate(ctx, messages)
		if err != nil {
			a.log.ERROR.Println(err)
			return Result{}, err
		}
		if resp == nil {
			return Result{}, errors.New("empty response")
		}

		reasoning := strings.TrimSpace(resp.ReasoningContent)
		a.log.TRACE.Printf("round %d: %d chars of reasoning", round+1, len(reasoning))

		answer := strings.TrimSpace(resp.Content)
		if answer != "" {
			content = answer
		}

		if len(resp.ToolCalls) == 0 {
			addStep(Step{Reasoning: reasoning})

			// a model that puts everything into its reasoning has not answered yet
			if answer == "" {
				return incomplete("empty answer")
			}

			// zero rounds means the model answered from its own knowledge
			a.log.DEBUG.Printf("answer after %d tool rounds", round)

			return Result{Content: answer, Steps: steps}, nil
		}

		calls := roundCalls(resp.ToolCalls)
		a.log.DEBUG.Printf("tool round %d: %v", round+1, callSignatures(calls))
		addStep(Step{Reasoning: reasoning, Calls: calls})

		// repeating a call once is a retry, doing it over and over is a model stuck in
		// a loop. Empty calls are not compared, they equal the nil of the first round.
		if len(calls) > 0 && slices.Equal(calls, previous) {
			if repeats++; repeats >= maxRepeats {
				return incomplete("repeated tool calls")
			}
		} else {
			repeats = 0
		}
		previous = calls

		messages = append(messages, schema.AssistantMessage(resp.Content, resp.ToolCalls))

		for _, tc := range resp.ToolCalls {
			messages = append(messages, schema.ToolMessage(
				truncateResult(a.callTool(ctx, tc)), tc.ID, schema.WithToolName(tc.Function.Name)))
		}
	}

	return incomplete(fmt.Sprintf("exceeded %d tool call rounds", maxIterations))
}

// truncateResult bounds a single tool result. Unbounded results push the tool
// definitions out of a small model's context, see maxHistory.
func truncateResult(s string) string {
	if len(s) <= maxToolResult {
		return s
	}
	return s[:maxToolResult] + "\n[truncated, ask for less at a time]"
}

// callTool executes a tool call and returns its textual result. Errors are returned to the model, not to the caller.
func (a *Assistant) callTool(ctx context.Context, tc schema.ToolCall) string {
	name := tc.Function.Name
	if name == "" {
		a.mcpLog.ERROR.Println("missing function call")
		return "error: missing function call"
	}

	var args map[string]any
	if s := strings.TrimSpace(tc.Function.Arguments); s != "" {
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
