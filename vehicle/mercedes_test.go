package vehicle

import (
	"fmt"
	"testing"
)

func TestNewMercedes(t *testing.T) {
	creds := map[string]interface{}{
		"ClientID":     "c144925f-8f3a-4140-99a0-afcd175d488f",
		"ClientSecret": "xwkBfLYUyvQBapCqmIfIiaCVOVfuvraPlppdhSSdcbxrICFbxmQwqvFdCuLDShoy",
	}

	mc, err := NewMercedesFromConfig(creds)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(mc.SoC())
}
