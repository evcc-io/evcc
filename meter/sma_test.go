package meter

import (
	"testing"

	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

func TestSMAUpdatePower(t *testing.T) {
	tests := []struct {
		name      string
		messsage  sma.Telegram
		wantPower float64
	}{
		{
			"success export",
			sma.Telegram{
				Values: map[string]float64{
					"1:1.4.0": 0,
					"1:2.4.0": 37.9,
				},
			},
			-37.9,
		},
		{
			"success import",
			sma.Telegram{
				Values: map[string]float64{
					"1:1.4.0": 20,
					"1:2.4.0": 0,
				},
			},
			20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SMA{
				log: util.NewLogger("sma "),
			}

			sm.updatePower(tt.messsage)
			if sm.power != tt.wantPower {
				t.Errorf("Listener.processMessage() got %v, want %v", sm.power, tt.wantPower)
			}
		})
	}
}
