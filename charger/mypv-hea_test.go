package charger

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sync"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type heaHandler struct {
	mbserver.DummyHandler
	sync.Mutex
	regs       map[uint16]uint16
	failWrites bool
}

func (h *heaHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.Lock()
	defer h.Unlock()

	if req.UnitId != 1 || req.Quantity != 1 {
		return nil, mbserver.ErrIllegalDataValue
	}
	if req.IsWrite {
		if req.Addr != 1080 {
			return nil, mbserver.ErrIllegalDataAddress
		}
		if h.failWrites || req.Args[0] > 7 {
			return nil, mbserver.ErrIllegalDataValue
		}
		h.regs[req.Addr] = req.Args[0]
	}
	value, ok := h.regs[req.Addr]
	if !ok {
		return nil, mbserver.ErrIllegalDataAddress
	}
	return []uint16{value}, nil
}

func TestMyPvHea(t *testing.T) {
	old := sponsor.Subject
	sponsor.Subject = ""
	t.Cleanup(func() { sponsor.Subject = old })
	_, err := NewMyPvHea(t.Context(), "test", modbus.TcpSettings{}, 1, 1)
	require.ErrorIs(t, err, api.ErrSponsorRequired)
	sponsor.Subject = "test"

	h := &heaHandler{regs: make(map[uint16]uint16)}
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	srv, err := mbserver.New(h)
	require.NoError(t, err)
	require.NoError(t, srv.Start(l))
	// Do not call mbserver.Stop: it races the accept loop's listener access.

	setRegisters := func(regs map[uint16]uint16) {
		h.Lock()
		defer h.Unlock()
		h.regs = regs
		h.failWrites = false
	}

	for _, tc := range []struct {
		name   string
		relays uint16
		step   float64
	}{
		{"mypv-hea-35", 1, 3500},
		{"mypv-hea-90", 3, 3000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			charger, err := NewFromConfig(t.Context(), tc.name, map[string]any{"uri": l.Addr().String()})
			require.NoError(t, err)
			wb := charger.(*MyPvHea)
			assert.Equal(t, uint16(1001), wb.regTemp)
			assert.Equal(t, tc.relays, wb.relays)
			current, err := wb.GetMaxCurrent()
			require.NoError(t, err)
			assert.Zero(t, current)

			minPower, maxPower, err := wb.GetMinMaxPower()
			require.NoError(t, err)
			assert.Equal(t, tc.step, minPower)
			assert.Equal(t, tc.step*float64(tc.relays), maxPower)

			assertMask := func(t *testing.T, want uint16) {
				t.Helper()
				b, err := wb.conn.ReadHoldingRegisters(1080, 1)
				require.NoError(t, err)
				assert.Equal(t, want, binary.BigEndian.Uint16(b))
			}

			t.Run("stages", func(t *testing.T) {
				for steps := uint16(1); steps <= tc.relays; steps++ {
					for _, delta := range []float64{-0.1, 0, 0.1} {
						power := float64(steps)*tc.step + delta
						wantSteps := steps
						if delta < 0 {
							wantSteps--
						}
						t.Run(fmt.Sprint(power), func(t *testing.T) {
							current := power / (voltage * float64(tc.relays))
							require.NoError(t, wb.MaxCurrentMillis(current))
							assertMask(t, (1<<wantSteps)-1)
							got, err := wb.GetMaxCurrent()
							require.NoError(t, err)
							assert.Equal(t, current, got)
						})
					}
				}
				require.NoError(t, wb.MaxCurrentMillis(0))
				assertMask(t, 0)
				require.NoError(t, wb.MaxCurrent(16))
				assertMask(t, (1<<tc.relays)-1)
				require.NoError(t, wb.MaxCurrentMillis(math.MaxFloat64))
				assertMask(t, (1<<tc.relays)-1)
			})

			t.Run("enable", func(t *testing.T) {
				require.NoError(t, wb.MaxCurrentMillis(tc.step/(voltage*float64(tc.relays))))
				require.NoError(t, wb.Enable(false))
				assertMask(t, 0)
				enabled, err := wb.Enabled()
				require.NoError(t, err)
				assert.False(t, enabled)
				require.NoError(t, wb.Enable(true))
				assertMask(t, 1)
				enabled, err = wb.Enabled()
				require.NoError(t, err)
				assert.True(t, enabled)
				current, err := wb.GetMaxCurrent()
				require.NoError(t, err)
				assert.Equal(t, tc.step/(voltage*float64(tc.relays)), current)
			})

			t.Run("invalid current", func(t *testing.T) {
				require.NoError(t, wb.MaxCurrentMillis(tc.step/(voltage*float64(tc.relays))))
				for _, current := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
					assert.Error(t, wb.MaxCurrentMillis(current))
					assertMask(t, 1)
					assert.Equal(t, uint32(1), wb.mask.Load())
					got, err := wb.GetMaxCurrent()
					require.NoError(t, err)
					assert.Equal(t, tc.step/(voltage*float64(tc.relays)), got)
				}
			})

			t.Run("write failure", func(t *testing.T) {
				require.NoError(t, wb.MaxCurrentMillis(tc.step/(voltage*float64(tc.relays))))
				h.Lock()
				h.failWrites = true
				h.Unlock()
				assert.Error(t, wb.MaxCurrent(0))
				assert.Error(t, wb.Enable(false))
				assert.Equal(t, uint32(1), wb.mask.Load())
				current, err := wb.GetMaxCurrent()
				require.NoError(t, err)
				assert.Equal(t, tc.step/(voltage*float64(tc.relays)), current)
				assertMask(t, 1)
				setRegisters(map[uint16]uint16{1080: 0})
				require.NoError(t, wb.Enable(true))
				assertMask(t, 1)
			})

			t.Run("enabled mask", func(t *testing.T) {
				for _, mask := range []uint16{0, 1, 2, 4, 7, 8, 15} {
					setRegisters(map[uint16]uint16{1080: mask})
					enabled, err := wb.Enabled()
					require.NoError(t, err)
					assert.Equal(t, mask&((1<<tc.relays)-1) != 0, enabled)
				}
			})

			t.Run("phases", func(t *testing.T) {
				lp := loadpoint.NewMockAPI(gomock.NewController(t))
				wb.LoadpointControl(lp)
				for _, phases := range []int{0, 1, 3} {
					lp.EXPECT().GetPhases().Return(phases)
					if phases == 0 {
						phases = int(tc.relays)
					}
					require.NoError(t, wb.MaxCurrentMillis(tc.step/(voltage*float64(phases))))
					assertMask(t, 1)
				}
				wb.LoadpointControl(nil)
			})

			t.Run("status", func(t *testing.T) {
				for _, tc := range []struct {
					state, power uint16
					want         api.ChargeStatus
				}{
					{0, 0, api.StatusB},
					{1, 3000, api.StatusC},
					{2, 6000, api.StatusC},
					{3, 0, api.StatusB},
					{3, 3500, api.StatusC},
					{1, 10, api.StatusB},
					{4, 0, api.StatusNone},
					{5, 3000, api.StatusNone},
					{6, 0, api.StatusNone},
				} {
					setRegisters(map[uint16]uint16{1077: tc.state, 1000: tc.power})
					status, err := wb.Status()
					assert.Equal(t, tc.want, status)
					if tc.want == api.StatusNone {
						assert.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				}
			})

			t.Run("measurements", func(t *testing.T) {
				setRegisters(map[uint16]uint16{1000: 3456, 1001: 456, 1002: 605, 1030: 512})
				power, err := wb.CurrentPower()
				require.NoError(t, err)
				assert.Equal(t, 3456.0, power)
				temp, err := wb.Soc()
				require.NoError(t, err)
				assert.Equal(t, 45.6, temp)
				limit, err := wb.GetLimitSoc()
				require.NoError(t, err)
				assert.Equal(t, int64(60), limit)
				charger, err := NewFromConfig(t.Context(), tc.name, map[string]any{
					"uri": l.Addr().String(), "tempsource": 2,
				})
				require.NoError(t, err)
				temp, err = charger.(*MyPvHea).Soc()
				require.NoError(t, err)
				assert.Equal(t, 51.2, temp)
			})

			t.Run("read failure", func(t *testing.T) {
				setRegisters(map[uint16]uint16{})
				_, err := wb.Status()
				assert.Error(t, err)
				_, err = wb.Enabled()
				assert.Error(t, err)
				_, err = wb.CurrentPower()
				assert.Error(t, err)
				_, err = wb.Soc()
				assert.Error(t, err)
				_, err = wb.GetLimitSoc()
				assert.Error(t, err)
				setRegisters(map[uint16]uint16{1077: 1})
				_, err = wb.Status()
				assert.Error(t, err)
			})
		})
	}

	t.Run("invalid config", func(t *testing.T) {
		for _, source := range []int{0, 3} {
			_, err := NewMyPvHea(t.Context(), "test", modbus.TcpSettings{}, source, 1)
			assert.ErrorContains(t, err, "invalid temp source")
		}
		_, err := NewMyPvHea(t.Context(), "test", modbus.TcpSettings{}, 1, 2)
		assert.ErrorContains(t, err, "invalid relay count")
		_, err = NewMyPvHea(t.Context(), "test", modbus.TcpSettings{}, 1, 1)
		assert.ErrorContains(t, err, "invalid modbus configuration")
		_, err = NewFromConfig(t.Context(), "mypv-hea-35", map[string]any{"unknown": true})
		assert.Error(t, err)
	})
}
