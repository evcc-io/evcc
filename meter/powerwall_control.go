package meter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/andig/go-powerwall"
	"github.com/evcc-io/evcc/api"
)

func anyFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported value type %T", v)
	}
}

func minPositive(values ...float64) float64 {
	var min float64
	for _, v := range values {
		if v > 0 && (min == 0 || v < min) {
			min = v
		}
	}
	return min
}

func (m *PowerWall) setBatteryMode(mode api.BatteryMode) error {
	switch mode {
	case api.BatteryNormal:
		return m.setReserve(m.minSoc)
	case api.BatteryHold:
		return m.holdReserve()
	case api.BatteryCharge:
		return m.setReserve(m.chargeMaxSoc)
	default:
		return errInvalidBatteryMode(mode)
	}
}

func (m *PowerWall) setReserve(limit float64) error {
	if limit < 0 || limit > 100 {
		return fmt.Errorf("invalid reserve: %v, must be 0-100", limit)
	}

	defer m.resetOperation()

	return m.postControl("/control/reserve", map[string]any{
		"value": int(math.Round(limit)),
	})
}

// reserve_hold has no idempotency key: if the response is lost, a retry may
// resolve a newer SoC and raise the reserve.
func (m *PowerWall) holdReserve() error {
	defer m.resetOperation()

	return m.postControl("/control/reserve_hold", map[string]any{})
}

func (m *PowerWall) resetOperation() {
	if m.opReset != nil {
		m.opReset()
	}
}

func (m *PowerWall) postControl(path string, body any) error {
	if m.controlSecret == "" {
		return fmt.Errorf("missing controlSecret")
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.controlBase+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.controlSecret)

	resp, err := m.controlClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("post %s: %s", path, resp.Status)
	}

	return nil
}

func (m *PowerWall) getControl(path string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.controlBase+path, nil)
	if err != nil {
		return err
	}

	resp, err := m.controlClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: %s", path, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (m *PowerWall) getFreq() (pwsFreq, error) {
	var res pwsFreq
	err := m.getControl("/freq", &res)
	return res, err
}

func (m *PowerWall) getOperation() (*powerwall.OperationData, error) {
	var res struct {
		BackupReservePercent *float64 `json:"backup_reserve_percent"`
	}

	if err := m.getControl("/api/operation", &res); err != nil {
		return nil, err
	}
	if res.BackupReservePercent == nil {
		return nil, fmt.Errorf("server reports no backup reserve")
	}

	return &powerwall.OperationData{BackupReservePercent: *res.BackupReservePercent}, nil
}

func (m *PowerWall) currents() (float64, float64, float64, error) {
	res, err := m.meterG()
	if err != nil {
		return 0, 0, 0, err
	}

	o, ok := res[m.usage]
	if !ok {
		return 0, 0, 0, fmt.Errorf("invalid usage: %s", m.usage)
	}

	return float64(o.IACurrent), float64(o.IBCurrent), float64(o.ICCurrent), nil
}
