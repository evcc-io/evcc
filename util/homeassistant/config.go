package homeassistant

import "github.com/evcc-io/evcc/util"

// Config holds the common Home Assistant instance connection parameters.
// It is embedded into device configs via mapstructure squash.
type Config struct {
	URI      string
	Token_   string `mapstructure:"token"` // TODO deprecated
	Home_    string `mapstructure:"home"`  // TODO deprecated
	Insecure bool   // optional - allow self-signed certificates
}

// NewConnection creates a Home Assistant connection from the config
func (c Config) NewConnection(log *util.Logger) (*Connection, error) {
	return NewConnection(log, c.URI, c.Home_, c.Insecure)
}
