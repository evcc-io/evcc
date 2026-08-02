package assistant

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
)

// ollamaDefaultUrl is the local ollama server
const ollamaDefaultUrl = "http://localhost:11434"

// ollamaUrl points a configured server url at the OpenAI-compatible endpoint
func ollamaUrl(base string) string {
	if base == "" {
		base = ollamaDefaultUrl
	}

	base = strings.TrimSuffix(base, "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}

	return base
}

// newLLM creates the language model for the configured provider
func newLLM(cfg Config) (llms.Model, error) {
	switch cfg.Provider {
	case OpenAI, Custom:
		// local OpenAI-compatible servers ignore the token but it must not be empty
		opts := []openai.Option{openai.WithModel(cfg.Model), openai.WithToken(cmp.Or(cfg.Token, "-"))}
		if cfg.BaseUrl != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseUrl))
		}
		return openai.New(opts...)

	case Anthropic:
		opts := []anthropic.Option{anthropic.WithModel(cfg.Model), anthropic.WithToken(cfg.Token)}
		if cfg.BaseUrl != "" {
			opts = append(opts, anthropic.WithBaseURL(cfg.BaseUrl))
		}
		return anthropic.New(opts...)

	case Ollama:
		// the native ollama binding drops the tools, the OpenAI-compatible endpoint
		// serves them. Its token is ignored but must not be empty.
		return openai.New(openai.WithModel(cfg.Model), openai.WithToken("ollama"),
			openai.WithBaseURL(ollamaUrl(cfg.BaseUrl)))

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
}
