package plugin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	solarmanv5 "github.com/evcc-io/evcc/plugin/solarman"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
)

func init() {
	registry.AddCtx("solarman", NewSolarmanFromConfig)
}

// Solarman reads Modbus registers through a Solarman V5 logger.
type Solarman struct {
	client *solarmanv5.Client
	id     byte
	reg    modbus.Register
	scale  float64
}

// NewSolarmanFromConfig creates a Solarman V5 plugin.
func NewSolarmanFromConfig(_ context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Host     string
		Port     int
		Serial   uint32
		Id       int
		Register modbus.Register
		Scale    float64
		Timeout  time.Duration
	}{
		Port:    8899,
		Id:      1,
		Scale:   1,
		Timeout: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if err := cc.Register.Error(); err != nil {
		return nil, err
	}
	if cc.Id < 1 || cc.Id > math.MaxUint8 {
		return nil, fmt.Errorf("invalid Modbus ID: %d", cc.Id)
	}

	function, err := cc.Register.FuncCode()
	if err != nil {
		return nil, err
	}
	if function != gridx.FuncCodeReadHoldingRegisters && function != gridx.FuncCodeReadInputRegisters {
		return nil, errors.New("Solarman supports holding and input registers only")
	}

	client, err := solarmanv5.New(cc.Host, cc.Port, cc.Serial, cc.Timeout)
	if err != nil {
		return nil, err
	}

	return &Solarman{
		client: client,
		id:     byte(cc.Id),
		reg:    cc.Register,
		scale:  cc.Scale,
	}, nil
}

func (s *Solarman) read(op modbus.RegisterOperation) ([]byte, error) {
	switch op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		return s.client.ReadHoldingRegisters(context.Background(), s.id, op.Addr, op.Length)
	case gridx.FuncCodeReadInputRegisters:
		return s.client.ReadInputRegisters(context.Background(), s.id, op.Addr, op.Length)
	default:
		return nil, fmt.Errorf("invalid read function code: %d", op.FuncCode)
	}
}

var _ FloatGetter = (*Solarman)(nil)

// FloatGetter implements func() (float64, error).
func (s *Solarman) FloatGetter() (func() (float64, error), error) {
	op, err := s.reg.Operation()
	if err != nil {
		return nil, err
	}

	decode, err := s.reg.DecodeFunc()
	if err != nil {
		return nil, err
	}

	return func() (float64, error) {
		bytes, err := s.read(op)
		if err != nil {
			return 0, fmt.Errorf("read failed: %w", err)
		}

		return s.scale * decode(bytes), nil
	}, nil
}

var _ IntGetter = (*Solarman)(nil)

// IntGetter implements func() (int64, error).
func (s *Solarman) IntGetter() (func() (int64, error), error) {
	getter, err := s.FloatGetter()
	if err != nil {
		return nil, err
	}

	return func() (int64, error) {
		value, err := getter()
		return int64(math.Round(value)), err
	}, nil
}
