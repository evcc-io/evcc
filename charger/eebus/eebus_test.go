package eebus

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	err := yaml.Unmarshal([]byte(conf), &res)
	require.NoError(t, err)
}
