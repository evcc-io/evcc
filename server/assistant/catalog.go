package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// The full tool set costs more context than a local model has to spare. Instead of
// offering every tool, the model searches a catalog and dispatches by name, which
// keeps the offered tools constant no matter how many exist.
const (
	findToolsName = "findTools"
	callToolName  = "callTool"

	// maxFindResults limits the tools described per search
	maxFindResults = 5
)

// directTools are offered next to the meta tools. Reading the system state and the
// documentation answers most questions, offering them directly saves a search round
// trip and keeps the model from inventing configuration advice.
var directTools = []string{"getState", "fetchDocs"}

// metaTools reach every tool that is not offered directly
var metaTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name: findToolsName,
			Description: "Search the tools that read and control this evcc system, for anything " +
				"the tools above cannot answer. Returns the name, description and parameters of " +
				"the matching tools. Search e.g. for 'charging mode', 'battery' or 'plan', then " +
				"call the tool you found with " + callToolName + ".",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "What you are looking for, e.g. 'charging mode'",
					},
				},
				"required": []string{"query"},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        callToolName,
			Description: "Call one of the tools returned by " + findToolsName + ".",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Name of the tool to call",
					},
					"arguments": map[string]any{
						"type":        "object",
						"description": "Arguments as described by the tool",
					},
				},
				"required": []string{"name"},
			},
		},
	},
}

// offeredTools returns the tools the model sees: the meta tools plus the few that
// are worth their context, the catalog holds the rest
func offeredTools(catalog []llms.Tool) []llms.Tool {
	res := slices.Clone(metaTools)

	for _, tool := range catalog {
		if slices.Contains(directTools, tool.Function.Name) {
			res = append(res, tool)
		}
	}

	return res
}

// findTools describes the catalog entries matching the query, name matches first
func (a *Assistant) findTools(args map[string]any) string {
	query, _ := args["query"].(string)
	terms := strings.Fields(strings.ToLower(query))

	var byName, byDescription []llms.Tool

	for _, tool := range a.catalog {
		name := strings.ToLower(tool.Function.Name)

		switch {
		case matchesAll(name, terms):
			byName = append(byName, tool)
		case matchesAll(name+" "+strings.ToLower(tool.Function.Description), terms):
			byDescription = append(byDescription, tool)
		}
	}

	res := append(byName, byDescription...)
	if len(res) == 0 {
		return fmt.Sprintf("no tool matches %q, try other words or fewer of them", query)
	}

	var sb strings.Builder
	for _, tool := range res[:min(len(res), maxFindResults)] {
		fmt.Fprintf(&sb, "%s: %s\n", tool.Function.Name, tool.Function.Description)

		// without the schema the model cannot build the arguments for callTool
		if params, err := json.Marshal(tool.Function.Parameters); err == nil {
			fmt.Fprintf(&sb, "parameters: %s\n", params)
		}

		sb.WriteString("\n")
	}

	if rest := len(res) - maxFindResults; rest > 0 {
		fmt.Fprintf(&sb, "%d more tools match, refine the query to see them.", rest)
	}

	return strings.TrimSpace(sb.String())
}

// matchesAll reports whether s contains all terms
func matchesAll(s string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(s, term) {
			return false
		}
	}
	return true
}

// dispatchArgs unpacks the tool the model wants to call
func dispatchArgs(args map[string]any) (string, map[string]any, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return "", nil, errors.New("missing tool name")
	}

	switch v := args["arguments"].(type) {
	case nil:
		return name, nil, nil

	case map[string]any:
		return name, v, nil

	case string:
		// models that cannot nest objects send the arguments as json string
		if strings.TrimSpace(v) == "" {
			return name, nil, nil
		}

		var res map[string]any
		if err := json.Unmarshal([]byte(v), &res); err != nil {
			return "", nil, fmt.Errorf("invalid arguments: %w", err)
		}

		return name, res, nil

	default:
		return "", nil, fmt.Errorf("invalid arguments: %T", v)
	}
}
