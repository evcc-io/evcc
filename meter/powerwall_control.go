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
	"strings"
	"time"

	"github.com/andig/go-powerwall"
	"github.com/evcc-io/evcc/api"
)

func anyFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
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

	if err := m.postControl("/control/reserve", map[string]any{
		"value": int(math.Round(limit)),
	}); err != nil {
		return err
	}

	m.resetOperation()
	return nil
}

func (m *PowerWall) holdReserve() error {
	if err := m.postControl("/control/reserve_hold", map[string]any{}); err != nil {
		return err
	}

	m.resetOperation()
	return nil
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(m.controlBase, "/")+path, bytes.NewReader(data))
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

func (m *PowerWall) getFreq() (pwsFreq, error) {
	var res pwsFreq

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(m.controlBase, "/")+"/freq", nil)
	if err != nil {
		return res, err
	}

	resp, err := m.controlClient.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("get /freq: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	return res, err
}

func (m *PowerWall) getOperation() (*powerwall.OperationData, error) {
	var res struct {
		BackupReservePercent *float64 `json:"backup_reserve_percent"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(m.controlBase, "/")+"/api/operation", nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.controlClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get /api/operation: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
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
