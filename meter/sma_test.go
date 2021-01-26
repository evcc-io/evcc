package meter

import (
	"testing"

	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

func TestSMAUpdateMeterValues(t *testing.T) {
	tests := []struct {
		name          string
		message       sma.Telegram
		wantPower     float64
		wantCurrentL1 float64
		wantCurrentL2 float64
		wantCurrentL3 float64
	}{
		{
			"success export",
			sma.Telegram{
				Values: map[string]float64{
					"1:1.4.0":  0,
					"1:2.4.0":  37.9,
					"1:31.4.0": 2.549,
					"1:51.4.0": 0.397,
					"1:71.4.0": 0.614,
				},
			},
			-37.9,
			2.549,
			0.397,
			0.614,
		},
		{
			"success import",
			sma.Telegram{
				Values: map[string]float64{
					"1:1.4.0":  20,
					"1:2.4.0":  0,
					"1:31.4.0": 0.654,
					"1:51.4.0": 0.245,
					"1:71.4.0": 0.231,
				},
			},
			20,
			0.654,
			0.245,
			0.231,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SMA{
				log: util.NewLogger("foo"),
				mux: util.NewWaiter(udpTimeout, func() {}),
			}

			sm.updateMeterValues(tt.message)
			if sm.values.power != tt.wantPower {
				t.Errorf("Listener.processMessage() got Power %v, want %v", sm.values.power, tt.wantPower)
			}

			if sm.values.currentL1 != tt.wantCurrentL1 {
				t.Errorf("Listener.processMessage() got CurrentL1 %v, want %v", sm.values.currentL1, tt.wantCurrentL1)
			}

			if sm.values.currentL2 != tt.wantCurrentL2 {
				t.Errorf("Listener.processMessage() got CurrentL2 %v, want %v", sm.values.currentL2, tt.wantCurrentL2)
			}

			if sm.values.currentL3 != tt.wantCurrentL3 {
				t.Errorf("Listener.processMessage() got CurrentL3 %v, want %v", sm.values.currentL3, tt.wantCurrentL3)
			}

		})
	}
}
