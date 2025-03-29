package charger

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/sirupsen/logrus"
	"github.com/spali/go-rscp/rscp"
	"github.com/spf13/cast"
)

type E3dc struct {
	conn    *rscp.Client
	id      uint8
	current byte
}

func init() {
	registry.Add("e3dc-rscp", NewE3dcFromConfig)
}

func NewE3dcFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Uri      string
		User     string
		Password string
		Key      string
		Id       uint8
		Timeout  time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	host, port_, err := net.SplitHostPort(util.DefaultPort(cc.Uri, 5033))
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(port_)

	cfg := rscp.ClientConfig{
		Address:           host,
		Port:              uint16(port),
		Username:          cc.User,
		Password:          cc.Password,
		Key:               cc.Key,
		ConnectionTimeout: cc.Timeout,
		SendTimeout:       cc.Timeout,
		ReceiveTimeout:    cc.Timeout,
	}

	return NewE3dc(cfg, cc.Id)
}

var e3dcOnce sync.Once

func NewE3dc(cfg rscp.ClientConfig, id uint8) (api.Charger, error) {
	e3dcOnce.Do(func() {
		log := util.NewLogger("e3dc")
		rscp.Log.SetLevel(logrus.DebugLevel)
		rscp.Log.SetOutput(log.TRACE.Writer())
	})

	conn, err := rscp.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	m := &E3dc{
		conn:    conn,
		id:      id,
		current: 6,
	}

	return m, nil
}

// Enabled implements the api.Charger interface
func (wb *E3dc) Enabled() (bool, error) {
	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALG, nil),
	}))
	if err != nil {
		return false, err
	}
	fmt.Printf("%+v\n", res)

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return false, err
	}

	wb_ext_data_alg, err := rscpContainer(wb_data[1], 2)
	if err != nil {
		return false, err
	}

	b, err := rscpBytes(wb_ext_data_alg[1])
	if err != nil {
		return false, err
	}

	return b[0]^(1<<4) == 0, err
}

// Enable implements the api.Charger interface
func (wb *E3dc) Enable(enable bool) error {
	return wb.maxCurrent(wb.current, enable)
}

// Status implements the api.Charger interface
func (wb *E3dc) Status() (api.ChargeStatus, error) {
	status := api.StatusNone

	res, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_REQ_DATA, []rscp.Message{
		*rscp.NewMessage(rscp.WB_INDEX, wb.id),
		*rscp.NewMessage(rscp.WB_REQ_EXTERN_DATA_ALL, nil),
	}))
	if err != nil {
		return status, err
	}
	fmt.Printf("%+v\n", res)

	wb_data, err := rscpContainer(*res, 2)
	if err != nil {
		return status, err
	}

	wb_ext_data_all, err := rscpContainer(wb_data[1], 2)
	if err != nil {
		return status, err
	}

	b, err := rscpBytes(wb_ext_data_all[1])
	if err != nil {
		return status, err
	}

	switch {
	case b[3] == 1:
		status = api.StatusC
	case b[4] == 1:
		status = api.StatusB
	default:
		status = api.StatusA
	}

	return status, nil
}

func (wb *E3dc) maxCurrent(current byte, enable bool) error {
	data := []rscp.Message{
		*rscp.NewMessage(rscp.WB_EXTERN_DATA, []byte{0x02, current, 0, 0, cast.ToUint8(!enable), 0}),
	}

	_, err := wb.conn.Send(*rscp.NewMessage(rscp.WB_SET_EXTERN, data))
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *E3dc) MaxCurrent(current int64) error {
	enabled, err := wb.Enabled()
	if err != nil {
		return err
	}

	err = wb.maxCurrent(byte(current), enabled)
	if err == nil {
		wb.current = byte(current)
	}

	return err
}

// var _ api.Meter = (*E3dc)(nil)

// // CurrentPower implements the api.Meter interface
// func (wb *E3dc) CurrentPower() (float64, error) {
// 	return 0, api.ErrNotAvailable
// }

// var _ api.ChargeRater = (*E3dc)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (wb *E3dc) ChargedEnergy() (float64, error) {
// 	return 0, api.ErrNotAvailable
// }

// var _ api.MeterEnergy = (*E3dc)(nil)

// // TotalEnergy implements the api.MeterEnergy interface
// func (wb *E3dc) TotalEnergy() (float64, error) {
// 	return 0, api.ErrNotAvailable
// }

// var _ api.PhaseCurrents = (*E3dc)(nil)

// // Currents implements the api.PhaseCurrents interface
// func (wb *E3dc) Currents() (float64, float64, float64, error) {
// 	return 0, 0, 0, api.ErrNotAvailable
// }

// var _ api.Identifier = (*E3dc)(nil)

// // Identify implements the api.Identifier interface
// func (wb *E3dc) Identify() (string, error) {
// 	return "", api.ErrNotAvailable
// }

// var _ api.PhaseSwitcher = (*E3dc)(nil)

// // Phases1p3p implements the api.PhaseSwitcher interface
// func (wb *E3dc) Phases1p3p(phases int) error {
// 	return api.ErrNotAvailable
// }

func rscpError(msg ...rscp.Message) error {
	var errs []error
	for _, m := range msg {
		if m.DataType == rscp.Error {
			errs = append(errs, errors.New(rscp.RscpError(cast.ToUint32(m.Value)).String()))
		}
	}
	return errors.Join(errs...)
}

func rscpContainer(msg rscp.Message, length int) ([]rscp.Message, error) {
	if err := rscpError(msg); err != nil {
		return nil, err
	}

	if msg.DataType != rscp.Container {
		return nil, errors.New("invalid response")
	}

	res, ok := msg.Value.([]rscp.Message)
	if !ok {
		return nil, errors.New("invalid response")
	}

	if l := len(res); l < length {
		return nil, fmt.Errorf("invalid length: expected %d, got %d", length, l)
	}

	return res, nil
}

func rscpBytes(msg rscp.Message) ([]byte, error) {
	return rscpValue(msg, func(data any) ([]byte, error) {
		b, ok := data.([]uint8)
		if !ok {
			return nil, errors.New("invalid response")
		}
		return b, nil
	})
}

func rscpValue[T any](msg rscp.Message, fun func(any) (T, error)) (T, error) {
	var zero T
	if err := rscpError(msg); err != nil {
		return zero, err
	}

	return fun(msg.Value)
}

// func rscpValues[T any](msg []rscp.Message, fun func(any) (T, error)) ([]T, error) {
// 	res := make([]T, 0, len(msg))

// 	for _, m := range msg {
// 		v, err := rscpValue(m, fun)
// 		if err != nil {
// 			return nil, err
// 		}

// 		res = append(res, v)
// 	}

// 	return res, nil
// }
