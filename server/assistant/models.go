package assistant

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	openaiUrl    = "https://api.openai.com/v1"
	anthropicUrl = "https://api.anthropic.com/v1"

	// anthropicVersion is required on every anthropic request
	anthropicVersion = "2023-06-01"
)

// modelsResponse is the shape all four providers answer /models with
type modelsResponse struct {
	Data []struct {
		Id string `json:"id"`
	} `json:"data"`
}

// modelsRequest builds the provider's model listing request
func modelsRequest(ctx context.Context, cfg Config) (*http.Request, error) {
	base := strings.TrimSuffix(cfg.BaseUrl, "/")
	headers := make(map[string]string)

	switch cfg.Provider {
	case OpenAI, Custom:
		if base == "" {
			base = openaiUrl
		}
		headers["Authorization"] = "Bearer " + cfg.Token

	case Anthropic:
		if base == "" {
			base = anthropicUrl
		}
		headers["x-api-key"] = cfg.Token
		headers["anthropic-version"] = anthropicVersion

	case Ollama:
		base = ollamaUrl(cfg.BaseUrl)

	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	return request.New(http.MethodGet, base+"/models", nil, headers)
}

// Models lists the models the configured endpoint offers
func Models(ctx context.Context, cfg Config) ([]string, error) {
	req, err := modelsRequest(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var res modelsResponse
	if err := request.NewHelper(util.NewLogger("assistant")).DoJSON(req.WithContext(ctx), &res); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(res.Data))
	for _, m := range res.Data {
		if m.Id != "" {
			models = append(models, m.Id)
		}
	}

	return models, nil
}
