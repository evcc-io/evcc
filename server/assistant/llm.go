package assistant

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// newLLM creates the language model for the configured provider
func newLLM(cfg Config) (llms.Model, error) {
	switch cfg.Provider {
	case OpenAI, Custom:
		opts := []openai.Option{openai.WithModel(cfg.Model), openai.WithToken(cfg.Token)}
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
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		if cfg.BaseUrl != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseUrl))
		}
		return ollama.New(opts...)

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
}
