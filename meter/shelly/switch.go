package shelly

type Switch struct {
	conn *Connection
}

func NewSwitch(conn *Connection) *Switch {
	res := &Switch{
		conn: conn,
	}

	return res
}
