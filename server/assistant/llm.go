package assistant

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// ollamaDefaultUrl is the local ollama server
const ollamaDefaultUrl = "http://localhost:11434"

// llmTimeout covers a slow model working through a tool loop
const llmTimeout = 2 * time.Minute

// anthropicMaxTokens bounds an answer, the api requires a limit
const anthropicMaxTokens = 4096

// llmClient traces the model requests like any other device connection
func llmClient() *http.Client {
	return &http.Client{
		Timeout:   llmTimeout,
		Transport: request.NewTripper(util.NewLogger("assistant"), transport.Default()),
	}
}

// azurePath is the OpenAI-compatible v1 endpoint of a Foundry resource, it
// needs no api version
const azurePath = "/openai/v1"

// azureHost reduces a configured resource or project url to its origin, the
// portal shows either and neither is the inference endpoint
func azureHost(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base url: %s", base)
	}

	return u.Scheme + "://" + u.Host, nil
}

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
func newLLM(ctx context.Context, cfg Config) (model.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case OpenAI, Custom:
		// local OpenAI-compatible servers ignore the token but it must not be empty
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			APIKey:     cmp.Or(cfg.Token, "-"),
			BaseURL:    cfg.BaseUrl,
			HTTPClient: llmClient(),
		})

	case Azure:
		// the v1 endpoint speaks plain OpenAI, the deployment name is the model
		host, err := azureHost(cfg.BaseUrl)
		if err != nil {
			return nil, err
		}

		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			APIKey:     cfg.Token,
			BaseURL:    host + azurePath,
			HTTPClient: llmClient(),
		})

	case Anthropic:
		cc := &claude.Config{
			Model:      cfg.Model,
			APIKey:     cfg.Token,
			MaxTokens:  anthropicMaxTokens,
			HTTPClient: llmClient(),
		}
		if cfg.BaseUrl != "" {
			cc.BaseURL = &cfg.BaseUrl
		}

		return claude.NewChatModel(ctx, cc)

	case Ollama:
		// the native ollama binding drops the tools, the OpenAI-compatible endpoint
		// serves them. Its token is ignored but must not be empty.
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			APIKey:     "ollama",
			BaseURL:    ollamaUrl(cfg.BaseUrl),
			HTTPClient: llmClient(),
		})

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
}
