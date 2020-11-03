package autoconf

import "github.com/andig/evcc/util"

// Detect detects configuration from given base config
func Detect(other map[string]interface{}) error {
	cc := struct {
		Broker   string
		User     string
		Password string
		Topic    string
	}{
		Topic: "openWB",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return err
	}

	return DetectOpenWB(cc.Broker, cc.User, cc.Password, cc.Topic)
}
