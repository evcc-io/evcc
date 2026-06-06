package plugin

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/evcc-io/evcc/plugin/aa55"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.AddCtx("aa55udp", NewAA55UDPFromConfig)
}

// NewAA55UDPFromConfig creates a GoodWe AA55-over-UDP plugin.
//
// Register read mode (single register):
//
//	source:   aa55udp
//	host:     192.168.1.26
//	id:       127          # 0x7F for DT/DNS/ES/EM (default); 247 (0xF7) for ET/EH/BT/BH
//	register: 30127
//	count:    2            # 1 = 16-bit, 2 = 32-bit
//	decode:   int32be
//	scale:    1.0
//
// Block read mode (fetch an enclosing block once, extract the target register):
//
//	source:   aa55udp
//	host:     192.168.1.26   # inverter IP; port 8899 is always used
//	id:       247            # 0x7F (default) for DT/DNS/ES/EM; 247 (0xF7) for ET/EH/BT/BH
//	register: 35139          # target register
//	count:    2              # 1 = 16-bit, 2 = 32-bit
//	block:                   # enclosing block fetched in a single UDP exchange
//	  register: 35100        # block start register
//	  count:    125          # block length (registers)
//	decode:   int32be        # int32be | uint32be | uint32nan | int16be | uint16be | float32be
//	scale:    1.0            # optional multiplier (default 1.0)
func NewAA55UDPFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Host     string
		Id       int
		Register uint16
		Count    uint16
		Block    *aa55.Block
		Decode   string
		Scale    float64
	}{
		Id:    int(aa55.InverterAddr),
		Count: 2,
		Scale: 1.0,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	raddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(cc.Host, "8899"))
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: request.Timeout}
	conn, err := dialer.DialUDP(ctx, "udp4", netip.AddrPort{}, raddr.AddrPort())
	if err != nil {
		return nil, err
	}

	res, err := aa55.New(util.NewLogger("aa55udp"), conn, cc.Id, cc.Register, cc.Count, cc.Block, cc.Decode, cc.Scale)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	return res, nil
}
