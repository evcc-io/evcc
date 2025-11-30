package eebus

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestConfig(t *testing.T) {
	conf := `
certificate:
  private: |
    -----BEGIN EC PRIVATE KEY-----
    MHcCfoo==
    -----END EC PRIVATE KEY-----
  public: |
    -----BEGIN CERTIFICATE-----
    MIIBbar=
    -----END CERTIFICATE-----
`

	var res Config
	require.NoError(t, yaml.Unmarshal([]byte(conf), &res))
}
