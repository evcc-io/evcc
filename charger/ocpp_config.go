package charger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (c *OCPP) getConfiguration(id string, connector int) (string, int, error) {
	var meterValuesSampledData string
	meterValuesSampledDataMaxLength := len(strings.Split(desiredMeasurands, ","))

	rc := make(chan error, 1)

	err := ocpp.Instance().GetConfiguration(id, func(resp *core.GetConfigurationConfirmation, err error) {
		if err == nil {
			for _, opt := range resp.ConfigurationKey {
				if opt.Value == nil {
					continue
				}

				switch opt.Key {
				case ocpp.KeyChargeProfileMaxStackLevel:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						c.stackLevel = val
					}

				case ocpp.KeyChargingScheduleAllowedChargingRateUnit:
					if *opt.Value == "Power" || *opt.Value == "W" { // "W" is not allowed by spec but used by some CPs
						c.chargingRateUnit = types.ChargingRateUnitWatts
					}

				case ocpp.KeyConnectorSwitch3to1PhaseSupported:
					var val bool
					if val, err = strconv.ParseBool(*opt.Value); err == nil {
						c.phaseSwitching = val
					}

				case ocpp.KeyMaxChargingProfilesInstalled:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						c.chargingProfileId = val
					}

				case ocpp.KeyMeterValuesSampledData:
					if opt.Readonly {
						meterValuesSampledDataMaxLength = 0
					}
					meterValuesSampledData = *opt.Value

				case ocpp.KeyMeterValuesSampledDataMaxLength:
					if val, err := strconv.Atoi(*opt.Value); err == nil {
						meterValuesSampledDataMaxLength = val
					}

				case ocpp.KeyNumberOfConnectors:
					var val int
					if val, err = strconv.Atoi(*opt.Value); err == nil && connector > val {
						err = fmt.Errorf("connector %d exceeds max available connectors: %d", connector, val)
					}

				case ocpp.KeySupportedFeatureProfiles:
					if !c.hasProperty(*opt.Value, smartcharging.ProfileName) {
						err = fmt.Errorf("the mandatory SmartCharging profile is not supported")
					}
					c.hasRemoteTriggerFeature = c.hasProperty(*opt.Value, remotetrigger.ProfileName)

				// vendor-specific keys
				case ocpp.KeyAlfenPlugAndChargeIdentifier:
					if c.idtag == defaultIdTag {
						c.idtag = *opt.Value
						c.log.DEBUG.Printf("overriding default `idTag` with Alfen-specific value: %s", c.idtag)
					}
				}

				if err != nil {
					break
				}
			}
		}

		rc <- err
	}, nil)

	return meterValuesSampledData, meterValuesSampledDataMaxLength, c.wait(err, rc)
}
