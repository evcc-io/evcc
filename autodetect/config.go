package autodetect

import (
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
)

// Detect detects configuration from given base config
func Detect(other map[string]interface{}) (*core.Site, error) {
	cc := struct {
		Broker   string
		User     string
		Password string
		Topic    string
	}{
		Broker: "localhost:1883",
		Topic:  "openWB",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return DetectOpenWB(cc.Broker, cc.User, cc.Password, cc.Topic)
}
