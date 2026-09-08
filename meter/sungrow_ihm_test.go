package meter

import (
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSungrowIHMBatteryControl(t *testing.T) {
	for _, tc := range []struct {
		name       string
		power      int
		mode       api.BatteryMode
		ems, cmd   uint16
		fail       uint16
		ignoreMode bool
		wantErr    bool
	}{
		{name: "charge", power: 6000, mode: api.BatteryCharge, ems: 1, cmd: 0xBB},
		{name: "lower power", power: 3500, mode: api.BatteryCharge, ems: 1, cmd: 0xAA},
		{name: "high word", power: 6553600, mode: api.BatteryCharge, ems: 1, cmd: 0xCC},
		{name: "refresh", power: 6000, mode: api.BatteryCharge, ems: 4, cmd: 0xAA},
		{name: "stale discharge", power: 6000, mode: api.BatteryCharge, ems: 4, cmd: 0xBB},
		{name: "hold", power: 6000, mode: api.BatteryHold, ems: 4, cmd: 0xAA},
		{name: "normal", power: 6000, mode: api.BatteryNormal, ems: 4, cmd: 0xAA},
		{name: "power write failure", power: 6000, mode: api.BatteryCharge, ems: 1, cmd: 0xBB, fail: 8025, wantErr: true},
		{name: "heartbeat write failure", power: 6000, mode: api.BatteryCharge, ems: 1, cmd: 0xBB, fail: 8032, wantErr: true},
		{name: "mode write failure", power: 6000, mode: api.BatteryCharge, ems: 1, cmd: 0xBB, fail: 8023, wantErr: true},
		{name: "mode ignored", power: 6000, mode: api.BatteryCharge, ems: 1, cmd: 0xBB, ignoreMode: true, wantErr: true},
		{name: "unsupported mode", power: 6000, mode: api.BatteryDischarge, ems: 1, cmd: 0xCC, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &sungrowIHMHandler{
				registers:  map[uint16]uint16{8023: tc.ems, 8024: tc.cmd},
				fail:       tc.fail,
				ignoreMode: tc.ignoreMode,
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			srv, err := mbserver.New(h)
			require.NoError(t, err)
			require.NoError(t, srv.Start(listener))
			t.Cleanup(func() { require.NoError(t, srv.Stop()) })

			m, err := NewFromTemplateConfig(t.Context(), map[string]any{
				"template":       "sungrow-ihm",
				"usage":          "battery",
				"modbus":         "tcpip",
				"host":           "127.0.0.1",
				"port":           listener.Addr().(*net.TCPAddr).Port,
				"maxchargepower": tc.power,
			})
			require.NoError(t, err)
			ctrl, ok := api.Cap[api.BatteryController](m)
			require.True(t, ok, "battery control must be available when maxchargepower is configured")
			assert.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}, ctrl.BatteryModes())
			t.Cleanup(func() {
				h.mu.Lock()
				h.fail = 0
				h.mu.Unlock()
				require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal))
			})

			err = ctrl.SetBatteryMode(tc.mode)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			h.mu.Lock()
			defer h.mu.Unlock()
			if tc.wantErr {
				assert.NotEqual(t, uint16(0xAA), h.registers[8024])
				return
			}
			wantMode, wantCmd := uint16(4), uint16(0xCC)
			if tc.mode == api.BatteryNormal {
				wantMode = 1
			} else {
				assert.Equal(t, uint16(60), h.registers[8032])
			}
			if tc.mode == api.BatteryCharge {
				wantCmd = 0xAA
				assert.Equal(t, uint16(tc.power/100), h.registers[8025])
				assert.Equal(t, uint16((tc.power/100)>>16), h.registers[8026])
			}
			assert.Equal(t, wantMode, h.registers[8023])
			assert.Equal(t, wantCmd, h.registers[8024])
			firstCmd := uint16(0xCC)
			if tc.name == "refresh" {
				firstCmd = 0xAA
			}
			require.NotEmpty(t, h.writes)
			assert.Equal(t, sungrowIHMWrite{8024, []uint16{firstCmd}}, h.writes[0])
			if tc.mode == api.BatteryCharge {
				assert.Equal(t, []sungrowIHMWrite{
					{8024, []uint16{firstCmd}},
					{8025, []uint16{uint16(tc.power / 100), uint16((tc.power / 100) >> 16)}},
					{8032, []uint16{60}},
					{8023, []uint16{4}},
					{8024, []uint16{0xAA}},
				}, h.writes)
			}
		})
	}
}

func TestSungrowIHMConfig(t *testing.T) {
	for _, tc := range []struct {
		usage   string
		power   int64
		control bool
		wantErr bool
	}{
		{usage: "grid", power: 6000},
		{usage: "pv", power: 6000},
		{usage: "battery"},
		{usage: "battery", power: 6000, control: true},
		{usage: "battery", power: -100, wantErr: true},
		{usage: "battery", power: 99, wantErr: true},
		{usage: "battery", power: 429496729600, wantErr: true},
	} {
		m, err := NewFromTemplateConfig(t.Context(), map[string]any{
			"template": "sungrow-ihm", "usage": tc.usage,
			"modbus": "tcpip", "host": "127.0.0.1", "maxchargepower": tc.power,
		})
		if tc.wantErr {
			require.Error(t, err, tc)
			continue
		}
		require.NoError(t, err, tc)
		assert.Equal(t, tc.control, api.HasCap[api.BatteryController](m), tc)
	}
}

func TestSungrowIHMWatchdog(t *testing.T) {
	h := &sungrowIHMHandler{
		registers: map[uint16]uint16{8023: 1, 8024: 0xBB},
		heartbeat: make(chan struct{}, 10),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv, err := mbserver.New(h)
	require.NoError(t, err)
	require.NoError(t, srv.Start(listener))
	t.Cleanup(func() { require.NoError(t, srv.Stop()) })

	cfg, err := templates.RenderInstance(templates.Meter, map[string]any{
		"template": "sungrow-ihm", "usage": "battery", "maxchargepower": 6000,
		"modbus": "tcpip", "host": "127.0.0.1", "port": listener.Addr().(*net.TCPAddr).Port,
	})
	require.NoError(t, err)
	watchdog := cfg.Other["batterymode"].(map[string]any)
	assert.Equal(t, "60s", watchdog["timeout"])
	watchdog["timeout"] = "40ms"
	m, err := NewFromConfig(t.Context(), cfg.Type, cfg.Other)
	require.NoError(t, err)
	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal)) })

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))
	for range 2 {
		select {
		case <-h.heartbeat:
		case <-time.After(5 * time.Second):
			t.Fatal("missing VPP heartbeat refresh")
		}
	}
	require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal))
	h.mu.Lock()
	writes := len(h.writes)
	assert.Equal(t, uint16(1), h.registers[8023])
	assert.Equal(t, uint16(0xCC), h.registers[8024])
	h.mu.Unlock()

	<-time.After(100 * time.Millisecond)
	h.mu.Lock()
	defer h.mu.Unlock()
	assert.Len(t, h.writes, writes, "normal must cancel watchdog writes")
}

type sungrowIHMWrite struct {
	address uint16
	values  []uint16
}

type sungrowIHMHandler struct {
	mbserver.DummyHandler
	mu         sync.Mutex
	registers  map[uint16]uint16
	writes     []sungrowIHMWrite
	fail       uint16
	ignoreMode bool
	heartbeat  chan struct{}
}

func (h *sungrowIHMHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if req.UnitId != 247 {
		return nil, mbserver.ErrIllegalDataAddress
	}
	if req.IsWrite {
		if req.Addr == h.fail {
			return nil, mbserver.ErrIllegalDataAddress
		}
		h.writes = append(h.writes, sungrowIHMWrite{req.Addr, slices.Clone(req.Args)})
		for i, value := range req.Args {
			if req.Addr == 8023 && value == 4 && h.ignoreMode {
				continue
			}
			h.registers[req.Addr+uint16(i)] = value
		}
		if req.Addr == 8032 && h.heartbeat != nil {
			select {
			case h.heartbeat <- struct{}{}:
			default:
			}
		}
		return nil, nil
	}
	res := make([]uint16, req.Quantity)
	for i := range res {
		res[i] = h.registers[req.Addr+uint16(i)]
	}
	return res, nil
}
