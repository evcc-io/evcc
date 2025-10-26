package marstekvenusapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Sender is a marstek UDP sender
type Sender struct {
	log     *util.Logger
	addr    string
	conn    *net.UDPConn
	tracker *RequestTracker
}

// NewSender creates Marstek Open API UDP sender
func NewSender(log *util.Logger, addr string, tracker *RequestTracker) (*Sender, error) {
	addr = util.DefaultPort(addr, Port)
	raddr, err := net.ResolveUDPAddr("udp", addr)

	var conn *net.UDPConn
	if err == nil {
		conn, err = net.DialUDP("udp", nil, raddr)
	}

	c := &Sender{
		log:     log,
		addr:    addr,
		conn:    conn,
		tracker: tracker,
	}

	return c, err
}

// sends a message, returns the ID that was used and tracked
func (c *Sender) SendMtekReq(rtype RequestType, req interface{}) (int, error) {
	c.log.TRACE.Printf("sending %s", rtype)
	nextid := c.tracker.GetNextID()

	var nextreq interface{}

	switch rtype {
	case METHOD_BATTERY_STATUS:
		nextreq = BatGetStatusReq{
			RpcRequest: RpcRequest{
				Id:     nextid,
				Method: METHOD_BATTERY_STATUS,
			},
			Params: &GetStatusReqParams{
				Id: 0,
			},
		}
	case METHOD_GET_DEVICE:
		nextreq = GetDeviceReq{
			RpcRequest: RpcRequest{
				Id:     nextid,
				Method: METHOD_GET_DEVICE,
			},
			Params: &GetDeviceReqParams{
				BleMac: "0",
			},
		}
	case METHOD_ES_STATUS:
		nextreq = EsGetStatusReq{
			RpcRequest: RpcRequest{
				Id:     nextid,
				Method: METHOD_ES_STATUS,
			},
			Params: &GetStatusReqParams{
				Id: 0,
			},
		}
	case METHOD_ES_MODE:
		nextreq = EsGetModeReq{
			RpcRequest: RpcRequest{
				Id:     nextid,
				Method: METHOD_ES_MODE,
			},
			Params: &GetStatusReqParams{
				Id: 0,
			},
		}

	default:
		c.log.TRACE.Printf("unexpected requesttype to be sent (not implemented) %s\n", rtype)
		return nextid, api.ErrNotAvailable
	}
	payload, err := json.Marshal(nextreq)
	if err != nil {
		c.log.TRACE.Printf("Error marshaling request: %v\n", err)
		return nextid, fmt.Errorf("unexpected requesttype to be sent %s", rtype)
	}
	c.tracker.TrackRequest(nextid, rtype)
	c.Send(string(payload))

	return nextid, nil
}

// Send msg to receiver
func (c *Sender) Send(msg string) error {
	c.log.TRACE.Printf("send to %s %v", c.addr, msg)
	_, err := io.Copy(c.conn, strings.NewReader(msg))
	return err
}
