package ocpp

import (
	"strconv"
	"strings"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/samber/lo"
)

func (cp *CP) Setup(meterValues string, meterInterval time.Duration) error {
	if err := cp.ChangeAvailabilityRequest(0, core.AvailabilityTypeOperative); err != nil {
		cp.log.DEBUG.Printf("failed configuring availability: %v", err)
	}

	// auto configuration
	desiredMeasurands := "Power.Active.Import,Energy.Active.Import.Register,Current.Import,Voltage,Current.Offered,Power.Offered,SoC"

	// remove offending measurands from desired values
	if remove, ok := strings.CutPrefix(meterValues, "-"); ok {
		desiredMeasurands = strings.Join(lo.Without(strings.Split(desiredMeasurands, ","), strings.Split(remove, ",")...), ",")
		meterValues = ""
	}

	meterValuesSampledDataMaxLength := len(strings.Split(desiredMeasurands, ","))

	resp, err := cp.GetConfigurationRequest()
	if err != nil {
		return err
	}

	for _, opt := range resp.ConfigurationKey {
		if opt.Value == nil {
			continue
		}

		match := func(s string) bool {
			return strings.EqualFold(opt.Key, s)
		}

		switch {
		case match(KeyChargeProfileMaxStackLevel):
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.StackLevel = val
			}

		case match(KeyChargingScheduleAllowedChargingRateUnit):
			if *opt.Value == "Power" || *opt.Value == "W" { // "W" is not allowed by spec but used by some CPs
				cp.ChargingRateUnit = types.ChargingRateUnitWatts
			}

		case match(KeyConnectorSwitch3to1PhaseSupported) || match(KeyChargeAmpsPhaseSwitchingSupported):
			var val bool
			if val, err = strconv.ParseBool(*opt.Value); err == nil {
				cp.PhaseSwitching = val
			}

		case match(KeyMaxChargingProfilesInstalled):
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.ChargingProfileId = val
			}

		case match(KeyMeterValuesSampledData):
			if opt.Readonly {
				meterValuesSampledDataMaxLength = 0
			}
			cp.meterValuesSample = strings.Join(lo.Map(strings.Split(*opt.Value, ","), func(s string, _ int) string {
				return strings.Trim(s, "' ")
			}), ",")

		case match(KeyMeterValuesSampledDataMaxLength):
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				meterValuesSampledDataMaxLength = val
			}

		case match(KeyNumberOfConnectors):
			if val, err := strconv.Atoi(*opt.Value); err == nil {
				cp.NumberOfConnectors = val
			}

		case match(KeySupportedFeatureProfiles):
			if !hasProperty(*opt.Value, smartcharging.ProfileName) {
				cp.log.WARN.Printf("the required SmartCharging feature profile is not indicated as supported")
			}
			// correct the availability assumption of RemoteTrigger only in case of a valid looking FeatureProfile list
			if hasProperty(*opt.Value, core.ProfileName) {
				cp.HasRemoteTriggerFeature = hasProperty(*opt.Value, remotetrigger.ProfileName)
			}

		// vendor-specific keys
		case match(KeyAlfenPlugAndChargeIdentifier):
			cp.IdTag = *opt.Value
			cp.log.DEBUG.Printf("overriding default `idTag` with Alfen-specific value: %s", cp.IdTag)

		case match(KeyEvBoxSupportedMeasurands):
			if meterValues == "" {
				meterValues = *opt.Value
			}
		}
	}

	// see who's there
	if cp.HasRemoteTriggerFeature {
		if err := cp.TriggerMessageRequest(0, core.BootNotificationFeatureName); err != nil {
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
		if err := cp.ChangeConfigurationRequest(KeyMeterValuesSampledData, meterValues); err == nil || meterValues == "disable" {
			cp.meterValuesSample = meterValues
		} else {
			cp.log.WARN.Printf("failed configuring %s: %v", KeyMeterValuesSampledData, err)
		}
	}

	// trigger initial meter values
	if cp.HasRemoteTriggerFeature {
		if err := cp.TriggerMessageRequest(0, core.MeterValuesFeatureName); err == nil {
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
		if err := cp.ChangeConfigurationRequest(KeyMeterValueSampleInterval, strconv.Itoa(int(meterInterval.Seconds()))); err != nil {
			cp.log.WARN.Printf("failed configuring %s: %v", KeyMeterValueSampleInterval, err)
		}
	}

	// configure websocket ping interval
	if err := cp.ChangeConfigurationRequest(KeyWebSocketPingInterval, "30"); err != nil {
		cp.log.DEBUG.Printf("failed configuring %s: %v", KeyWebSocketPingInterval, err)
	}

	// trigger status for all connectors
	if cp.HasRemoteTriggerFeature {
		var ok bool

		// apply cached status if available
		instance.WithChargepointStatusByID(cp.id, func(status *core.StatusNotificationRequest) {
			if _, err := cp.OnStatusNotification(status); err == nil {
				ok = true
			}
		})

		// only trigger if we don't already have a status
		if !ok {
			if err := cp.TriggerMessageRequest(0, core.StatusNotificationFeatureName); err != nil {
				cp.log.WARN.Printf("failed triggering StatusNotification: %v", err)
			}
		}
	}

	return nil
}

// HasMeasurement checks if meterValuesSample contains given measurement
func (cp *CP) HasMeasurement(val types.Measurand) bool {
	return hasProperty(cp.meterValuesSample, string(val))
}

func (cp *CP) tryMeasurands(measurands string, key string) []string {
	var accepted []string
	for _, m := range strings.Split(measurands, ",") {
		if err := cp.ChangeConfigurationRequest(key, m); err == nil {
			accepted = append(accepted, m)
		}
	}
	return accepted
}
