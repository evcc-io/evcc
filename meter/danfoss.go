package meter

import (
	"context"
	"fmt"
	"strings"
	"time"

	comlynx "github.com/PanterSoft/comlynx-go"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
)

// DanfossTLX is a PV meter for Danfoss TripleLynx TLX inverters via ComLynx RS485.
type DanfossTLX struct {
	implement.Caps
	conn          *comlynx.Client
	powerFallback bool // some TLX variants don't support aggregate power; sum per-phase instead
}

func init() {
	registry.AddCtx("danfoss-tlx", NewDanfossTLXFromConfig)
}

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
		destination, err := parseComlynxNodeAddress(cc.Node)
		if err != nil {
			return nil, fmt.Errorf("node %q: %w", cc.Node, err)
		}
		cfg.Destination = destination
	}

	return NewDanfossTLX(ctx, cfg, cc.pvMaxACPower.Decorator())
}

func NewDanfossTLX(ctx context.Context, cfg comlynx.Config, maxACPower func() float64) (api.Meter, error) {
	log := util.NewLogger("danfoss-tlx")

	conn, err := comlynx.New(log.TRACE.Printf, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Destination == (comlynx.Address{}) {
		addr, err := comlynx.Discover(conn)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("address discovery: %w", err)
		}
		conn.SetDestination(addr)
		log.DEBUG.Printf("discovered inverter at %s", addr)
	}

	m := &DanfossTLX{
		Caps: implement.New(),
		conn: conn,
	}

	// probe capabilities
	_, aggregatePowerErr := conn.Read(comlynx.ParamGridPowerTotal)
	_, hasEnergy := conn.Read(comlynx.ParamTotalEnergy)
	_, _, _, hasVoltages := m.phaseVoltages()
	_, _, _, hasCurrents := m.phaseCurrents()
	_, _, _, hasPowers := m.phasePowers()

	if aggregatePowerErr != nil {
		if hasPowers == nil {
			m.powerFallback = true
		} else {
			_ = conn.Close()
			return nil, fmt.Errorf("power unavailable: aggregate read failed (%w) and per-phase powers are unavailable", aggregatePowerErr)
		}
	}

	if hasEnergy == nil {
		implement.Has(m, implement.MeterImport(m.totalEnergy))
	}
	if hasVoltages == nil {
		implement.Has(m, implement.PhaseVoltages(m.phaseVoltages))
	}
	if hasCurrents == nil {
		implement.Has(m, implement.PhaseCurrents(m.phaseCurrents))
	}
	if hasPowers == nil {
		implement.Has(m, implement.PhasePowers(m.phasePowers))
	}
	implement.May(m, implement.MaxACPowerGetter(maxACPower))

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	return m, nil
}

func (m *DanfossTLX) CurrentPower() (float64, error) {
	if m.powerFallback {
		p1, p2, p3, err := m.phasePowers()
		if err != nil {
			return 0, err
		}
		return p1 + p2 + p3, nil
	}
	v, err := m.conn.Read(comlynx.ParamGridPowerTotal)
	return float64(v), err
}

func (m *DanfossTLX) totalEnergy() (float64, error) {
	v, err := m.conn.Read(comlynx.ParamTotalEnergy)
	if err != nil {
		return 0, err
	}
	return float64(v) / 1000, nil // Wh → kWh
}

func (m *DanfossTLX) phaseVoltages() (float64, float64, float64, error) {
	return m.getPhases(comlynx.ParamGridVoltageL1, comlynx.ParamGridVoltageL2, comlynx.ParamGridVoltageL3, 10) // raw is V*10
}

func (m *DanfossTLX) phaseCurrents() (float64, float64, float64, error) {
	return m.getPhases(comlynx.ParamGridCurrentL1, comlynx.ParamGridCurrentL2, comlynx.ParamGridCurrentL3, 1000) // raw is mA
}

func (m *DanfossTLX) phasePowers() (float64, float64, float64, error) {
	return m.getPhases(comlynx.ParamGridPowerL1, comlynx.ParamGridPowerL2, comlynx.ParamGridPowerL3, 1)
}

func (m *DanfossTLX) getPhases(p1, p2, p3 uint16, divisor float64) (float64, float64, float64, error) {
	params := [3]uint16{p1, p2, p3}
	vals := [3]float64{}
	for i, p := range params {
		v, err := m.conn.Read(p)
		if err != nil {
			return 0, 0, 0, err
		}
		vals[i] = float64(v) / divisor
	}
	return vals[0], vals[1], vals[2], nil
}

func parseComlynxNodeAddress(value string) (comlynx.Address, error) {
	var network, subnet, node int
	if _, err := fmt.Sscanf(value, "%x-%x-%x", &network, &subnet, &node); err != nil {
		return comlynx.Address{}, fmt.Errorf("expected format N-S-NN in hex (e.g. c-6-b1): %w", err)
	}

	if network < 0 || network > 0x0f {
		return comlynx.Address{}, fmt.Errorf("network component %x out of range (must be 0..f)", network)
	}
	if subnet < 0 || subnet > 0x0f {
		return comlynx.Address{}, fmt.Errorf("subnet component %x out of range (must be 0..f)", subnet)
	}
	if node < 0 || node > 0xff {
		return comlynx.Address{}, fmt.Errorf("node component %x out of range (must be 00..ff)", node)
	}

	return comlynx.NewAddress(byte(network), byte(subnet), byte(node)), nil
}
