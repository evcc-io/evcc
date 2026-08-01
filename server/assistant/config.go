package assistant

import (
	"errors"
	"fmt"
	"slices"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
)

// Provider identifies the AI api flavour
type Provider string

const (
	OpenAI    Provider = "openai"
	Anthropic Provider = "anthropic"
	Ollama    Provider = "ollama"
	Custom    Provider = "custom" // any OpenAI-compatible endpoint
)

var providers = []Provider{OpenAI, Anthropic, Ollama, Custom}

// Config is the assistant configuration
type Config struct {
	Provider Provider `json:"provider"`
	Model    string   `json:"model"`
	Token    string   `json:"token,omitempty"`
	BaseUrl  string   `json:"baseUrl,omitempty"`
}

// Redacted implements the redactor interface used by the tee publisher
func (c Config) Redacted() any {
	c.Token = util.Masked(c.Token)
	return c
}

func (c Config) validate() error {
	if !slices.Contains(providers, c.Provider) {
		return fmt.Errorf("invalid provider: %s", c.Provider)
	}
	if c.Model == "" {
		return errors.New("missing model")
	}
	if c.Provider == Custom && c.BaseUrl == "" {
		return errors.New("missing base url")
	}
	if c.Provider != Ollama && c.Provider != Custom && c.Token == "" {
		return errors.New("missing token")
	}
	return nil
}

// ConfiguredConfig returns the persisted configuration
func ConfiguredConfig() (Config, error) {
	var cfg Config
	if !settings.Exists(keys.Assistant) {
		return cfg, errors.New("assistant not configured")
	}
	if err := settings.Json(keys.Assistant, &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.validate()
}
