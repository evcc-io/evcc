package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelsRequest(t *testing.T) {
	for _, tc := range []struct {
		cfg     Config
		url     string
		headers map[string]string
	}{
		{
			Config{Provider: OpenAI, Token: "t"},
			"https://api.openai.com/v1/models",
			map[string]string{"Authorization": "Bearer t"},
		},
		{
			Config{Provider: OpenAI, Token: "t", BaseUrl: "https://gateway.local/v1/"},
			"https://gateway.local/v1/models",
			map[string]string{"Authorization": "Bearer t"},
		},
		{
			Config{Provider: Anthropic, Token: "t"},
			"https://api.anthropic.com/v1/models",
			map[string]string{"X-Api-Key": "t", "Anthropic-Version": anthropicVersion},
		},
		{
			Config{Provider: Custom, Token: "t", BaseUrl: "http://localhost:1234/v1"},
			"http://localhost:1234/v1/models",
			map[string]string{"Authorization": "Bearer t"},
		},
		{
			// the server url is pointed at the OpenAI-compatible endpoint
			Config{Provider: Ollama, BaseUrl: "http://nas:11434"},
			"http://nas:11434/v1/models",
			nil,
		},
		{
			Config{Provider: Ollama},
			"http://localhost:11434/v1/models",
			nil,
		},
	} {
		req, err := modelsRequest(t.Context(), tc.cfg)
		require.NoError(t, err, tc.cfg)

		assert.Equal(t, tc.url, req.URL.String())
		for k, v := range tc.headers {
			assert.Equal(t, v, req.Header.Get(k), tc.cfg)
		}
	}

	_, err := modelsRequest(t.Context(), Config{Provider: "bogus"})
	assert.Error(t, err)
}
