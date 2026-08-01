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

// systemPrompt is embedded at build time, edit prompt.txt to change it
//
//go:embed prompt.txt
var systemPrompt string

// Message is a single chat message
type Message struct {
	Role    string `json:"role"` // user|assistant
	Content string `json:"content"`
}

// Assistant answers questions about the running system using evcc's MCP tools
type Assistant struct {
	log     *util.Logger
	mcpLog  *util.Logger
	llm     llms.Model
	client  *mcpsdk.ClientSession
	server  *mcpsdk.ServerSession
	tools   []llms.Tool
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

	tools, err := listTools(ctx, cs)
	if err != nil {
		cs.Close()
		ss.Close()
		return nil, err
	}

	return &Assistant{
		log: util.NewLogger("assistant"),
		// tool calls are mcp requests, log them with the mcp transport they trigger
		mcpLog: util.NewLogger("mcp"),
		llm:    llm,
		client: cs,
		server: ss,
		tools:  tools,
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

// Chat answers the conversation, calling tools as needed
func (a *Assistant) Chat(ctx context.Context, history []Message) (string, error) {
	prompt := strings.TrimSpace(systemPrompt)
	if a.context != "" {
		prompt += "\n\nContext for this conversation:\n" + a.context
	}

	messages := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, prompt)}

	for _, m := range history {
		role := llms.ChatMessageTypeHuman
		if m.Role == "assistant" {
			role = llms.ChatMessageTypeAI
		}
		messages = append(messages, llms.TextParts(role, m.Content))
	}

	for range maxIterations {
		resp, err := a.llm.GenerateContent(ctx, messages, llms.WithTools(a.tools))
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("empty response")
		}

		choice := resp.Choices[0]
		if len(choice.ToolCalls) == 0 {
			return choice.Content, nil
		}

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

	return "", fmt.Errorf("exceeded %d tool call rounds", maxIterations)
}

// callTool executes a tool call and returns its textual result. Errors are returned to the model, not to the caller.
func (a *Assistant) callTool(ctx context.Context, tc llms.ToolCall) string {
	if tc.FunctionCall == nil {
		a.mcpLog.ERROR.Println("missing function call")
		return "error: missing function call"
	}

	var args map[string]any
	if s := strings.TrimSpace(tc.FunctionCall.Arguments); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			a.mcpLog.ERROR.Printf("tool %s invalid arguments: %v", tc.FunctionCall.Name, err)
			return "error: invalid arguments: " + err.Error()
		}
	}

	a.mcpLog.DEBUG.Printf("tool %s %v", tc.FunctionCall.Name, args)

	res, err := a.client.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      tc.FunctionCall.Name,
		Arguments: args,
	})
	if err != nil {
		a.mcpLog.ERROR.Printf("tool %s: %v", tc.FunctionCall.Name, err)
		return "error: " + err.Error()
	}

	var sb strings.Builder
	for _, c := range res.Content {
		if txt, ok := c.(*mcpsdk.TextContent); ok {
			sb.WriteString(txt.Text)
		}
	}

	if res.IsError {
		a.mcpLog.ERROR.Printf("tool %s: %s", tc.FunctionCall.Name, sb.String())
		return "error: " + sb.String()
	}

	a.mcpLog.TRACE.Printf("tool %s result: %s", tc.FunctionCall.Name, sb.String())

	return sb.String()
}
