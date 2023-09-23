package ocpp

type ChargepointConnector struct {
	*CP
	*Connector
}

func NewChargepointConnector() *ChargepointConnector {
	return &ChargepointConnector{
		CP:        NewChargePoint(nil, "", 1, 0),
		Connector: NewConnector(),
	}
}
