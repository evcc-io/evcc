package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const desiredMeasurands = "Power.Active.Import,Energy.Active.Import.Register,Current.Import,Voltage,Current.Offered,Power.Offered,SoC"

func (cp *CP) Setup(meterValues string, meterInterval time.Duration) error {
	if err := Instance().ChangeAvailabilityRequest(cp.ID(), 0, core.AvailabilityTypeOperative); err != nil {
		cp.log.DEBUG.Printf("failed configuring availability: %v", err)
	}

	meterValuesSampledDataMaxLength := len(strings.Split(desiredMeasurands, ","))

	resp, err := cp.GetConfiguration()
	if err != nil {
		return err
	}

	for _, opt := range resp.ConfigurationKey {
		if opt.Value == nil {
			continue
		}

		switch opt.Key {
		case KeyChargeProfileMaxStackLevel:
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.StackLevel = val
			}

		case KeyChargingScheduleAllowedChargingRateUnit:
			if *opt.Value == "Power" || *opt.Value == "W" { // "W" is not allowed by spec but used by some CPs
				cp.ChargingRateUnit = types.ChargingRateUnitWatts
			}

		case KeyConnectorSwitch3to1PhaseSupported:
			var val bool
			if val, err = strconv.ParseBool(*opt.Value); err == nil {
				cp.PhaseSwitching = val
			}

		case KeyMaxChargingProfilesInstalled:
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.ChargingProfileId = val
			}

		case KeyMeterValuesSampledData:
			if opt.Readonly {
				meterValuesSampledDataMaxLength = 0
			}
			cp.meterValuesSample = *opt.Value

		case KeyMeterValuesSampledDataMaxLength:
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				meterValuesSampledDataMaxLength = val
			}

		case KeyNumberOfConnectors:
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.NumberOfConnectors = val
			}

		case KeySupportedFeatureProfiles:
			if !hasProperty(*opt.Value, smartcharging.ProfileName) {
				cp.log.WARN.Printf("the required SmartCharging feature profile is not indicated as supported")
			}
			// correct the availability assumption of RemoteTrigger only in case of a valid looking FeatureProfile list
			if hasProperty(*opt.Value, core.ProfileName) {
				cp.HasRemoteTriggerFeature = hasProperty(*opt.Value, remotetrigger.ProfileName)
			}

		// vendor-specific keys
		case KeyAlfenPlugAndChargeIdentifier:
			cp.IdTag = *opt.Value
			cp.log.DEBUG.Printf("overriding default `idTag` with Alfen-specific value: %s", cp.IdTag)

		case KeyEvBoxSupportedMeasurands:
			if meterValues == "" {
				meterValues = *opt.Value
			}
		}
	}

	// see who's there
	if cp.HasRemoteTriggerFeature {
		if err := Instance().TriggerMessageRequest(cp.ID(), core.BootNotificationFeatureName); err != nil {
			cp.log.DEBUG.Printf("failed triggering BootNotification: %v", err)
		}

		select {
		case <-time.After(Timeout):
			cp.log.DEBUG.Printf("BootNotification timeout")
		case res := <-cp.bootNotificationRequestC:
			cp.BootNotificationResult = res
		}
	}

	// autodetect measurands
	if meterValues == "" && meterValuesSampledDataMaxLength > 0 {
		sampledMeasurands := cp.tryMeasurands(desiredMeasurands, KeyMeterValuesSampledData)
		meterValues = strings.Join(sampledMeasurands[:min(len(sampledMeasurands), meterValuesSampledDataMaxLength)], ",")
	}

	// configure measurands
	if meterValues != "" {
		if err := cp.configure(KeyMeterValuesSampledData, meterValues); err == nil || meterValues == "disable" {
			cp.meterValuesSample = meterValues
		} else {
			cp.log.WARN.Printf("failed configuring %s: %v", KeyMeterValuesSampledData, err)
		}
	}

	// trigger initial meter values
	if cp.HasRemoteTriggerFeature {
		if err := Instance().TriggerMessageRequest(cp.ID(), core.MeterValuesFeatureName); err == nil {
			// wait for meter values
			select {
			case <-time.After(Timeout):
				cp.log.WARN.Println("meter timeout")
			case <-cp.meterC:
			}
		}
	}

	// configure sample rate
	if meterInterval > 0 {
		if err := cp.configure(KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
			cp.log.WARN.Printf("failed configuring %s: %v", KeyMeterValueSampleInterval, err)
		}
	}

	// configure websocket ping interval
	if err := cp.configure(KeyWebSocketPingInterval, "30"); err != nil {
		cp.log.DEBUG.Printf("failed configuring %s: %v", KeyWebSocketPingInterval, err)
	}

	return nil
}

// GetConfiguration
func (cp *CP) GetConfiguration() (*core.GetConfigurationConfirmation, error) {
	rc := make(chan error, 1)

	var res *core.GetConfigurationConfirmation
	err := Instance().GetConfiguration(cp.ID(), func(resp *core.GetConfigurationConfirmation, err error) {
		res = resp
		rc <- err
	}, nil)

	return res, wait(err, rc)
}

// HasMeasurement checks if meterValuesSample contains given measurement
func (cp *CP) HasMeasurement(val types.Measurand) bool {
	return hasProperty(cp.meterValuesSample, string(val))
}

func (cp *CP) tryMeasurands(measurands string, key string) []string {
	var accepted []string
	for _, m := range strings.Split(measurands, ",") {
		if err := cp.configure(key, m); err == nil {
			accepted = append(accepted, m)
		}
	}
	return accepted
}

// configure updates CP configuration
func (cp *CP) configure(key, val string) error {
	rc := make(chan error, 1)

	err := Instance().ChangeConfiguration(cp.id, func(resp *core.ChangeConfigurationConfirmation, err error) {
		if err == nil && resp != nil && resp.Status != core.ConfigurationStatusAccepted {
			rc <- fmt.Errorf("ChangeConfiguration failed: %s", resp.Status)
		}

		rc <- err
	}, key, val)

	return wait(err, rc)
}
