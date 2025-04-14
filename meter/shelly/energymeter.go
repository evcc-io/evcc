package shelly

type EnergyMeter struct {
	conn *Connection
}

func NewEnergyMeter(conn *Connection) *EnergyMeter {
	res := &EnergyMeter{
		conn: conn,
	}

	return res
}
