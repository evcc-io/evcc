package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/machine"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	api            = "https://api.evcc.io"
	enabledSetting = "telemetry"
)

var (
	instanceID string

	mu                              sync.Mutex
	updated                         time.Time
	accChargeEnergy, accGreenEnergy float64
)

func Enabled() bool {
	enabled, _ := settings.Bool(enabledSetting)
	return enabled && sponsor.IsAuthorizedForApi() && instanceID != ""
}

func Enable(enable bool) error {
	if enable {
		if !sponsor.IsAuthorized() {
			return errors.New("telemetry requires sponsorship")
		}
		if instanceID == "" {
			return fmt.Errorf("using docker? Telemetry requires a unique instance ID. Add this to your config: `plant: %s`", machine.RandomID())
		}
	}

	settings.SetBool(enabledSetting, enable)

	return nil
}

func Create(machineID string) {
	if machineID == "" {
		machineID, _ = machine.ProtectedID("evcc-api")
	}

	instanceID = machineID
}

// UpdateChargeProgress uploads power and energy data every 30 seconds
func UpdateChargeProgress(log *util.Logger, power, greenShare float64) {
	mu.Lock()
	defer mu.Unlock()

	if time.Since(updated) < 30*time.Second {
		return
	}

	if err := upload(log, power, power*greenShare); err != nil {
		log.ERROR.Printf("telemetry: upload failed: %v", err)
	}
}

// UpdateEnergy accumulates the energy delta for later upload
func UpdateEnergy(chargeEnergy, greenEnergy float64) {
	mu.Lock()
	defer mu.Unlock()

	// cache
	accChargeEnergy += chargeEnergy
	accGreenEnergy += greenEnergy
}

// Persist uploads the accumulated data if necessary
func Persist(log *util.Logger) {
	mu.Lock()
	defer mu.Unlock()

	if accChargeEnergy+accGreenEnergy == 0 {
		return
	}

	if err := upload(log, 0, 0); err != nil {
		log.ERROR.Printf("telemetry: upload failed: %v", err)
	}
}

// upload executes the actual upload.
// Lock must be held when calling upload.
func upload(log *util.Logger, chargePower, greenPower float64) error {
	log.TRACE.Printf("telemetry: charge: Δ%.0f/%.0fWh @ %.0fW", accGreenEnergy*1e3, accChargeEnergy*1e3, chargePower)

	data := InstanceChargeProgress{
		InstanceID: instanceID,
		ChargeProgress: ChargeProgress{
			ChargePower:  chargePower,
			GreenPower:   greenPower,
			ChargeEnergy: accChargeEnergy,
			GreenEnergy:  accGreenEnergy,
		},
	}

	uri := fmt.Sprintf("%s/v1/charge", api)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Authorization": "Bearer " + sponsor.Token,
	})

	// request timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	req = req.WithContext(ctx)
	defer cancel()

	if err == nil {
		client := request.NewHelper(log)

		var res struct {
			Error string
		}

		if err = client.DoJSON(req, &res); err == nil && res.Error != "" {
			err = errors.New(res.Error)
		}
	}

	if err == nil {
		updated = time.Now()

		accChargeEnergy = 0
		accGreenEnergy = 0
	}

	return err
}
