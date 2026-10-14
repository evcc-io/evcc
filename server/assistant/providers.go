package assistant

// ProviderInfo describes a provider to the configuration ui
type ProviderInfo struct {
	Provider Provider `json:"provider"`
	// Models are suggestions, the first one is the default. The field stays free
	// text, any model the endpoint offers may be entered
	Models []string `json:"models,omitempty"`
	// BaseUrl is the endpoint used when none is configured, shown as example
	BaseUrl string `json:"baseUrl,omitempty"`
	// NeedsToken and NeedsBaseUrl mark the fields the provider cannot do without
	NeedsToken   bool `json:"needsToken"`
	NeedsBaseUrl bool `json:"needsBaseUrl"`
}

var providers = []ProviderInfo{
	{
		Provider:   OpenAI,
		Models:     []string{"gpt-5", "gpt-5-mini", "gpt-4.1", "gpt-4o-mini"},
		NeedsToken: true,
	},
	{
		Provider:   Anthropic,
		Models:     []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"},
		NeedsToken: true,
	},
	{
		Provider: Ollama,
		Models: []string{
			"qwen3", "llama3.1", "llama3.2", "mistral-nemo",
			"gpt-oss", "command-r7b", "granite3.3", "firefunction-v2",
		},
		BaseUrl: ollamaDefaultUrl,
	},
	{
		Provider:     Custom,
		BaseUrl:      "https://generativelanguage.googleapis.com/v1beta/openai",
		NeedsBaseUrl: true,
	},
	{
		// the deployment names are the customer's own, there is nothing to suggest
		Provider:     Azure,
		BaseUrl:      "https://my-resource.services.ai.azure.com",
		NeedsToken:   true,
		NeedsBaseUrl: true,
	},
}

// Providers describes the supported providers to the configuration ui
func Providers() []ProviderInfo {
	return providers
}

// providerInfo returns the description of a provider
func providerInfo(p Provider) (ProviderInfo, bool) {
	for _, info := range providers {
		if info.Provider == p {
			return info, true
		}
	}

	return ProviderInfo{}, false
}
