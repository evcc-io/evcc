package assistant

import (
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

// azureApiVersion is the GA data plane version, recent enough for tool calling
const azureApiVersion = "2024-10-21"

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
		// a local OpenAI-compatible server needs no key, an empty one sends no header
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			APIKey:     cfg.Token,
			BaseURL:    cfg.BaseUrl,
			HTTPClient: llmClient(),
		})

	case Azure:
		// the model is the deployment name, it goes into the url
		host, err := azureHost(cfg.BaseUrl)
		if err != nil {
			return nil, err
		}

		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			APIKey:     cfg.Token,
			BaseURL:    host,
			ByAzure:    true,
			APIVersion: azureApiVersion,
			HTTPClient: llmClient(),
			// the configured deployment name is used as is, the default mapper
			// strips dots and would miss a deployment like gpt-5.6-luna
			AzureModelMapperFunc: func(model string) string { return model },
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
		// the native ollama binding drops the tools, the OpenAI-compatible
		// endpoint serves them and needs no key
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:      cfg.Model,
			BaseURL:    ollamaUrl(cfg.BaseUrl),
			HTTPClient: llmClient(),
		})

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
}
