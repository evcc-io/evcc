package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	comlynx "github.com/PanterSoft/comlynx-go"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// DanfossTLX is a PV meter for Danfoss TripleLynx TLX inverters via ComLynx RS485.
type DanfossTLX struct {
	conn          *comlynx.Client
	log           *util.Logger
	powerFallback bool // some TLX variants don't support aggregate power; sum per-phase instead
}

func init() {
	registry.AddCtx("danfoss-tlx", NewDanfossTLXFromConfig)
}

//go:generate go tool decorate -f decorateDanfossTLX -b *DanfossTLX -r api.Meter -t api.MeterEnergy,api.PhaseVoltages,api.PhaseCurrents,api.PhasePowers,api.MaxACPowerGetter

func NewDanfossTLXFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		pvMaxACPower `mapstructure:",squash"`
		Usage        string
		Device       string
		URI          string
		Baudrate     int
		Node         string
		Timeout      time.Duration
	}{
		Baudrate: comlynx.DefaultBaudrate,
		Timeout:  comlynx.DefaultTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if !strings.EqualFold(cc.Usage, "pv") {
		return nil, fmt.Errorf("danfoss-tlx only supports usage 'pv', got %q", cc.Usage)
	}

	cfg := comlynx.Config{
		Device:   cc.Device,
		URI:      cc.URI,
		Baudrate: cc.Baudrate,
		Timeout:  cc.Timeout,
		Source:   comlynx.DefaultSource,
	}

	if cc.Node != "" {
		var network, subnet, node int
		if _, err := fmt.Sscanf(cc.Node, "%x-%x-%x", &network, &subnet, &node); err != nil {
			return nil, fmt.Errorf("node %q: expected format N-S-NN in hex (e.g. c-6-b1)", cc.Node)
		}
		for _, component := range []struct {
			name  string
			value int
		}{
			{name: "network", value: network},
			{name: "subnet", value: subnet},
			{name: "node", value: node},
		} {
			if component.value < 0 || component.value > 0xff {
				return nil, fmt.Errorf("node %q: %s component %x out of range (must be 00..ff)", cc.Node, component.name, component.value)
			}
		}
		cfg.Destination = comlynx.NewAddress(byte(network), byte(subnet), byte(node))
	}

	return NewDanfossTLX(ctx, cfg, cc.pvMaxACPower.Decorator())
}

func NewDanfossTLX(ctx context.Context, cfg comlynx.Config, maxACPower func() float64) (api.Meter, error) {
	log := util.NewLogger("danfoss-tlx")

	conn, err := comlynx.New(log.TRACE.Printf, cfg)
	if err != nil {
		return nil, err
	}

	if (cfg.Destination == comlynx.Address{}) {
		addr, err := comlynx.Discover(conn)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("address discovery: %w", err)
		}
		conn.SetDestination(addr)
		log.DEBUG.Printf("discovered inverter at %s", addr)
	}

	m := &DanfossTLX{conn: conn, log: log}

	// Release the connection when the context is cancelled.
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	if _, err := conn.Read(comlynx.ParamGridPowerTotal); err != nil {
		log.WARN.Printf("aggregate power unavailable, falling back to per-phase sum: %v", err)
		m.powerFallback = true
	}

	var totalEnergy func() (float64, error)
	if _, err := conn.Read(comlynx.ParamTotalEnergy); err == nil {
		totalEnergy = m.totalEnergy
	}

	var voltages, currents, powers func() (float64, float64, float64, error)
	if ok := m.probePhase(comlynx.ParamGridVoltageL1, comlynx.ParamGridVoltageL2, comlynx.ParamGridVoltageL3); ok {
		voltages = m.phaseVoltages
	}
	if ok := m.probePhase(comlynx.ParamGridCurrentL1, comlynx.ParamGridCurrentL2, comlynx.ParamGridCurrentL3); ok {
		currents = m.phaseCurrents
	}
	if ok := m.probePhase(comlynx.ParamGridPowerL1, comlynx.ParamGridPowerL2, comlynx.ParamGridPowerL3); ok {
		powers = m.phasePowers
	}

	return decorateDanfossTLX(m, totalEnergy, voltages, currents, powers, maxACPower), nil
}

func (m *DanfossTLX) probePhase(p1, p2, p3 uint16) bool {
	for _, p := range []uint16{p1, p2, p3} {
		if _, err := m.conn.Read(p); err != nil {
			return false
		}
	}
	return true
}

func (m *DanfossTLX) CurrentPower() (float64, error) {
	if m.powerFallback {
		return m.sumPhasePowers()
	}
	v, err := m.conn.Read(comlynx.ParamGridPowerTotal)
	return float64(v), err
}

func (m *DanfossTLX) sumPhasePowers() (float64, error) {
	var total float64
	for _, p := range []uint16{comlynx.ParamGridPowerL1, comlynx.ParamGridPowerL2, comlynx.ParamGridPowerL3} {
		v, err := m.conn.Read(p)
		if err != nil {
			return 0, err
		}
		total += float64(v)
	}
	return total, nil
}

func (m *DanfossTLX) totalEnergy() (float64, error) {
	v, err := m.conn.Read(comlynx.ParamTotalEnergy)
	if err != nil {
		return 0, err
	}
	return float64(v) / 1000, nil // Wh → kWh
}

func (m *DanfossTLX) phaseVoltages() (float64, float64, float64, error) {
	return m.readThree(comlynx.ParamGridVoltageL1, comlynx.ParamGridVoltageL2, comlynx.ParamGridVoltageL3, 10) // raw is V*10
}

func (m *DanfossTLX) phaseCurrents() (float64, float64, float64, error) {
	return m.readThree(comlynx.ParamGridCurrentL1, comlynx.ParamGridCurrentL2, comlynx.ParamGridCurrentL3, 1000) // raw is mA
}

func (m *DanfossTLX) phasePowers() (float64, float64, float64, error) {
	return m.readThree(comlynx.ParamGridPowerL1, comlynx.ParamGridPowerL2, comlynx.ParamGridPowerL3, 1)
}

func (m *DanfossTLX) readThree(p1, p2, p3 uint16, divisor float64) (float64, float64, float64, error) {
	errs := make([]error, 0, 3)
	vals := make([]float64, 3)
	for i, p := range []uint16{p1, p2, p3} {
		v, err := m.conn.Read(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		vals[i] = float64(v) / divisor
	}
	return vals[0], vals[1], vals[2], errors.Join(errs...)
}
