package ocpp

import "github.com/evcc-io/evcc/util"

type Connector struct {
	log *util.Logger
	*CP
	connector int
}

func NewConnector(log *util.Logger, cp *CP, connector int) *Connector {
	return &Connector{
		log:       log,
		CP:        cp,
		connector: connector,
	}
}
