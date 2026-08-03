package assistant

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
)

// ollamaDefaultUrl is the local ollama server
const ollamaDefaultUrl = "http://localhost:11434"

// llmTimeout covers a slow model working through a tool loop
const llmTimeout = 2 * time.Minute

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
func newLLM(cfg Config) (llms.Model, error) {
	switch cfg.Provider {
	case OpenAI, Custom:
		// local OpenAI-compatible servers ignore the token but it must not be empty
		opts := []openai.Option{openai.WithModel(cfg.Model), openai.WithToken(cmp.Or(cfg.Token, "-")),
			openai.WithHTTPClient(llmClient())}
		if cfg.BaseUrl != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseUrl))
		}
		return openai.New(opts...)

	case Azure:
		// the v1 endpoint speaks plain OpenAI, the deployment name is the model
		host, err := azureHost(cfg.BaseUrl)
		if err != nil {
			return nil, err
		}
		return openai.New(openai.WithModel(cfg.Model), openai.WithToken(cfg.Token),
			openai.WithBaseURL(host+azurePath), openai.WithHTTPClient(llmClient()))

	case Anthropic:
		opts := []anthropic.Option{anthropic.WithModel(cfg.Model), anthropic.WithToken(cfg.Token),
			anthropic.WithHTTPClient(llmClient())}
		if cfg.BaseUrl != "" {
			opts = append(opts, anthropic.WithBaseURL(cfg.BaseUrl))
		}
		return anthropic.New(opts...)

	case Ollama:
		// the native ollama binding drops the tools, the OpenAI-compatible endpoint
		// serves them. Its token is ignored but must not be empty.
		return openai.New(openai.WithModel(cfg.Model), openai.WithToken("ollama"),
			openai.WithBaseURL(ollamaUrl(cfg.BaseUrl)), openai.WithHTTPClient(llmClient()))

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}
}
