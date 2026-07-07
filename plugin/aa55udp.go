package plugin

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/evcc-io/evcc/plugin/aa55"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.AddCtx("aa55udp", NewAA55UDPFromConfig)
}

// NewAA55UDPFromConfig creates a GoodWe AA55-over-UDP plugin. The register is
// configured as a modbus.Register; its count is derived from the decode width.
//
// Register read mode (single register):
//
//	source:   aa55udp
//	host:     192.168.1.26
//	id:       127            # 0x7F for DT/DNS/ES/EM (default); 247 (0xF7) for ET/EH/BT/BH
//	register:
//	  address: 30127
//	  decode:  int32be       # int32be | uint32be | uint32nan | int16be | uint16be | float32be
//	scale:    1.0
//
// Block read mode (fetch an enclosing block once, extract the target register):
//
//	source:   aa55udp
//	host:     192.168.1.26   # inverter IP; port 8899 is always used
//	id:       247            # 0x7F (default) for DT/DNS/ES/EM; 247 (0xF7) for ET/EH/BT/BH
//	register:
//	  address: 35139         # target register
//	  decode:  int32be
//	block:                   # enclosing block fetched in a single UDP exchange
//	  register: 35100        # block start register
//	  count:    125          # block length (registers)
//	scale:    1.0            # optional multiplier (default 1.0)
//	delay:    100ms          # optional min gap between sends to one inverter (0 disables)
func NewAA55UDPFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Host     string
		Id       int
		Register modbus.Register
		Block    *modbus.Block
		Scale    float64
		Delay    time.Duration
	}{
		Id:    int(aa55.InverterAddr),
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

	log := util.NewLogger("aa55udp")

	// write register type selects the setter, otherwise a (block) reader
	if strings.HasPrefix(strings.ToLower(cc.Register.Type), "write") {
		res, err := aa55.NewSetter(log, conn, cc.Id, cc.Register, cc.Scale, cc.Delay)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("aa55udp: %w", err)
		}
		return res, nil
	}

	res, err := aa55.New(log, conn, cc.Id, cc.Register, cc.Block, cc.Scale, cc.Delay)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	return res, nil
}
