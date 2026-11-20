package meter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlow represents the EcoFlow  meter
type EcoFlow struct {
	implement.Caps
	log     *util.Logger
	usage   string
	serials []string
	leader  atomic.Int32
	cache   time.Duration
	client  *ecoflow.Client
	dataG   func() (*ecoflow.GetCmdResponse, error)
	limitG  func() (float64, error)
	mainG   func() (string, error)
	mainSn  string
	known   map[string]bool

	power, batterySoc string
}

func init() {
	registry.Add("ecoflow", NewEcoFlowFromConfig)
}

// NewEcoFlowFromConfig creates an EcoFlow  meter from generic config
func NewEcoFlowFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                      `mapstructure:",squash"`
		batteryPowerLimits                   `mapstructure:",squash"`
		batterySocLimits                     `mapstructure:",squash"`
		Usage                                string
		AccessKey, SecretKey, Serial, Region string
		Power, Soc                           string
		MainDevice                           bool
		Cache                                time.Duration
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.AccessKey == "" {
		return nil, errors.New("missing access key")
	}
	if cc.SecretKey == "" {
		return nil, errors.New("missing secret key")
	}
	if cc.Serial == "" {
		return nil, errors.New("missing serial")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	var uri string
	switch cc.Region {
	case "auto":
		uri = "https://api.ecoflow.com"
	case "europe":
		uri = "https://api-e.ecoflow.com"
	case "america":
		uri = "https://api-a.ecoflow.com"
	default:
		return nil, fmt.Errorf("invalid region: %s", cc.Region)
	}

	var serials []string
	for sn := range strings.SplitSeq(cc.Serial, ",") {
		if sn = strings.TrimSpace(sn); sn != "" {
			serials = append(serials, sn)
		}
	}

	return NewEcoFlow(cc.AccessKey, cc.SecretKey, serials, cc.Usage, uri, cc.Power, cc.Soc, cc.MainDevice, cc.Cache, cc.batteryCapacity.Decorator(), cc.batterySocLimits, cc.batteryPowerLimits.Decorator())
}

// NewEcoFlow constructs the EcoFlow struct
func NewEcoFlow(accessKey, secretKey string, serials []string, usage, uri string,
	power, soc string, mainDevice bool, cache time.Duration, capacity func() (float64, error), batterySocLimits batterySocLimits, batteryPowerLimits func() (float64, float64, error)) (*EcoFlow, error) {
	if len(serials) == 0 {
		return nil, errors.New("missing serial")
	}

	log := util.NewLogger("ecoflow").Redact(append([]string{accessKey, secretKey}, serials...)...)
	client := request.NewClient(log)

	m := &EcoFlow{
		Caps:    implement.New(),
		log:     log,
		serials: serials,
		usage:   usage,
		cache:   cache,
		client: ecoflow.NewEcoflowClient(accessKey, secretKey,
			ecoflow.WithBaseUrl(uri),
			ecoflow.WithHttpClient(client),
		),
		power:      power,
		batterySoc: soc,
	}

	m.dataG = util.Cached(m.getData, cache)

	// only commands follow the main device, reads stay on the configured serial since summed per-device
	// meters rely on followers reading zero. A serial list keeps detecting the leader by cmsBattSoc.
	if mainDevice && len(serials) == 1 {
		m.known = map[string]bool{serials[0]: true}
		m.mainG = util.Cached(func() (string, error) {
			sn, err := m.mainSerial(ecoflow.NewHttpRequest(client, http.MethodGet, uri+"/iot-open/sign/device/system/main/sn",
				map[string]any{"sn": serials[0]}, accessKey, secretKey))
			if err != nil {
				// keep the last main device, the error lets the cache retry with backoff
				m.log.DEBUG.Printf("main device: %v", err)
				return m.mainSn, err
			}
			return sn, nil
		}, cache)
	}

	if usage == "battery" {
		implement.Has(m, implement.Battery(m.soc))
		implement.May(m, implement.BatteryCapacity(capacity))
		implement.May(m, implement.BatterySocLimiter(batterySocLimits.Decorator()))
		implement.May(m, implement.BatteryPowerLimiter(batteryPowerLimits))

		// the backup reserve command is Stream-specific, the PowerOcean template shares this meter type
		if soc == "cmsBattSoc" {
			m.limitG = util.Cached(m.dischargeLimit, cache)
			implement.Has(m, implement.BatteryController(batteryModesSocLimit, batterySocLimits.LimitController(m.holdSoc, m.setBackupReserve)))
		}
	}

	return m, nil
}

// ecoflowLeaderKey is only reported non-zero by the elected leader of a Stream system
const ecoflowLeaderKey = "cmsBattSoc"

// serial returns the serial commands are sent to: the main device if looked up, the current leader otherwise
func (m *EcoFlow) serial() string {
	if m.mainG != nil {
		if sn, _ := m.mainG(); sn != "" {
			return sn
		}
	}

	return m.serials[m.leader.Load()]
}

// mainSerial resolves the configured serial to the main device of its system
func (m *EcoFlow) mainSerial(req *ecoflow.HttpRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b, err := req.Execute(ctx)
	if err != nil {
		return "", err
	}

	var res struct {
		Code, Message string
		Data          struct{ Sn string }
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return "", err
	}
	if res.Code != "0" {
		return "", fmt.Errorf("%s (%s)", res.Message, res.Code)
	}
	if res.Data.Sn == "" {
		return "", errors.New("invalid response")
	}

	if !m.known[res.Data.Sn] {
		m.log.Redact(res.Data.Sn)
		m.known[res.Data.Sn] = true
	}
	m.mainSn = res.Data.Sn

	return res.Data.Sn, nil
}

func (m *EcoFlow) query(serial string, params []string) (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return m.client.GetDeviceParameters(ctx, serial, params)
}

// isLeader reports whether the response comes from the leader of a Stream system
func isLeader(res *ecoflow.GetCmdResponse) bool {
	v, err := ecoflowValue(res.Data, ecoflowLeaderKey)
	return err == nil && v > 0
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlow) getData() (*ecoflow.GetCmdResponse, error) {
	params := []string{m.power}

	if m.usage == "battery" {
		params = append(params, m.batterySoc)
	}

	if len(m.serials) > 1 && !slices.Contains(params, ecoflowLeaderKey) {
		params = append(params, ecoflowLeaderKey)
	}

	cur := int(m.leader.Load())

	res, err := m.query(m.serials[cur], params)
	if err != nil || len(m.serials) == 1 || isLeader(res) {
		return res, err
	}

	// leader may have been re-elected, followers report zero values
	for i, sn := range m.serials {
		if i == cur {
			continue
		}

		if r, err := m.query(sn, params); err == nil && isLeader(r) {
			m.log.DEBUG.Printf("leader changed to %s", sn)
			m.leader.Store(int32(i))
			return r, nil
		}
	}

	return res, nil
}

var _ api.Meter = (*EcoFlow)(nil)

// CurrentPower implements the api.Meter interface
func (m *EcoFlow) CurrentPower() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	pwr, err := ecoflowValue(response.Data, m.power)
	if err != nil {
		return 0, err
	}

	if m.usage == "battery" {
		pwr = -pwr // invert battery power: ecoflow returns negative when discharging and positive when charging.
	}

	return pwr, nil
}

// extractFloat extracts a float64 or int value from a map by key.
func ecoflowValue(data map[string]any, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return cast.ToFloat64E(v)
		}
	}
	return 0, api.ErrNotAvailable
}

// soc returns the battery state of charge
func (m *EcoFlow) soc() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return ecoflowValue(response.Data, m.batterySoc)
}

// holdSoc returns the soc of the device the backup reserve is sent to
func (m *EcoFlow) holdSoc() (float64, error) {
	sn := m.serial()
	if m.mainG == nil || sn == m.serials[0] {
		return m.soc()
	}

	res, err := m.query(sn, []string{m.batterySoc})
	if err != nil {
		return 0, err
	}

	return ecoflowValue(res.Data, m.batterySoc)
}

// ecoflowReserveLimit clamps the reserve above the discharge limit, which the device requires it
// to exceed by 3
func ecoflowReserveLimit(limit, lo float64) float64 {
	return min(100, max(limit, lo+3))
}

// dischargeLimit reads the device's discharge limit (cmsMinDsgSoc), set in the EcoFlow app and
// polled separately from getData so a rejected quota can't degrade the power and soc readings
func (m *EcoFlow) dischargeLimit() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := m.client.GetDeviceParameters(ctx, m.serial(), []string{"cmsMinDsgSoc"})
	if err != nil {
		return 0, err
	}

	return ecoflowValue(res.Data, "cmsMinDsgSoc")
}

func (m *EcoFlow) setBackupReserve(limit float64) error {
	lo, err := m.limitG()
	if err != nil {
		return fmt.Errorf("discharge limit: %w", err)
	}

	// device requires the reserve to exceed the discharge limit by 3
	if clamped := ecoflowReserveLimit(limit, lo); clamped != limit {
		m.log.DEBUG.Printf("backup reserve: raised %.0f to %.0f, below discharge limit %.0f", limit, clamped, lo)
		limit = clamped
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// cfgBackupReverseSoc is the vendor's spelling of the backup reserve
	res, err := m.client.SetDeviceParameter(ctx, map[string]any{
		"sn": m.serial(), "cmdId": 17, "cmdFunc": 254,
		"dirDest": 1, "dirSrc": 1, "dest": 2, "needAck": true,
		"params": map[string]any{"cfgBackupReverseSoc": int(math.Round(limit))},
	})
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("invalid response")
	}
	if res.Code != "0" {
		return fmt.Errorf("%s (%s)", res.Message, res.Code)
	}
	return nil
}
