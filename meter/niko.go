package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/niko"
	"github.com/evcc-io/evcc/util"
)

// Niko Home Control smart plug via Hobby API

func init() {
	registry.Add("niko", NewNikoFromConfig)
}

// NewNikoFromConfig creates a Niko Home Control smart plug meter from generic config
func NewNikoFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host   string
		User   string
		Token  string
		Device string
	}{
		User: "hobby",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return niko.NewConnection(cc.Host, cc.User, cc.Token, cc.Device)
}
