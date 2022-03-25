package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

const timeout = 2 * time.Minute

// txnCount is the global transaction id counter
var txnCount int64

type smartchargingChargeProfileKey string

// Smart Charging Profile Key
const (
	chargeProfileMaxStackLevel              smartchargingChargeProfileKey = "ChargeProfileMaxStackLevel"
	chargingScheduleAllowedChargingRateUnit smartchargingChargeProfileKey = "ChargingScheduleAllowedChargingRateUnit"
	chargingScheduleMaxPeriods              smartchargingChargeProfileKey = "ChargingScheduleMaxPeriods"
	connectorSwitch3to1PhaseSupported       smartchargingChargeProfileKey = "ConnectorSwitch3to1PhaseSupported"
	maxChargingProfilesInstalled            smartchargingChargeProfileKey = "MaxChargingProfilesInstalled"
)

type smartChargingProfile struct {
	// Max StackLevel of a ChargingProfile. The number defined also indicates the max allowed
	// number of installed charging scheduls per Charging Profile Purpose
	ChargeProfileMaxStackLevel int
	// A list of supported quantities for use in a ChargingSchedule.
	// Allowed values: 'Current' and 'Power'
	ChargingScheduleAllowedChargingRateUnit []string
	// Maximum number of periods that may be defined per ChargingSchedule
	ChargingScheduleMaxPeriods int
	// Defines if this Charge Point support switching from 3 to 1 phase during a charging session.
	ConnectorSwitch3to1PhaseSupported bool
	// Maximum number of Charging profiles instsalled at a time.
	MaxChargingProfilesInstalled int
}

type CP struct {
	mu  sync.Mutex
	log *util.Logger
	id  string
	txn int // current transaction

	updated     time.Time
	initialized *sync.Cond
	boot        *core.BootNotificationRequest
	status      *core.StatusNotificationRequest
	meterValues *core.MeterValuesRequest

	supportedNumberOfConnectors int
	smartChargingCapabilities   smartChargingProfile
}

func (cp *CP) DetectCapabilities(opts []core.ConfigurationKey) error {
	options := make(map[string]core.ConfigurationKey)
	for _, opt := range opts {
		options[opt.Key] = opt
	}

	{
		supported, err := parseIntOption("NumberOfConnectors", options)
		if err != nil {
			return err
		}

		cp.supportedNumberOfConnectors = supported
	}

	{ // meter detection
		// TODO
	}

	smartChargingCapabilities, err := detectSmartChargingCapabilities(options)
	if err != nil {
		return err
	}

	cp.smartChargingCapabilities = smartChargingCapabilities

	return nil
}

func (cp *CP) GetNumberOfSupportedConnectors() int {
	return cp.supportedNumberOfConnectors
}

func detectSmartChargingCapabilities(options map[string]core.ConfigurationKey) (smartChargingProfile, error) {
	var profile smartChargingProfile

	{ // required
		val, err := parseIntOption(chargeProfileMaxStackLevel, options)
		if err != nil {
			return profile, err
		}

		profile.ChargeProfileMaxStackLevel = val
	}

	{ // required
		val, err := parseIntOption(chargingScheduleMaxPeriods, options)
		if err != nil {
			return profile, err
		}

		profile.ChargingScheduleMaxPeriods = val
	}

	{ // required
		val, err := parseIntOption(maxChargingProfilesInstalled, options)
		if err != nil {
			return profile, err
		}

		profile.MaxChargingProfilesInstalled = val
	}

	{ // required
		opt, found := options[string(chargingScheduleAllowedChargingRateUnit)]
		if !found || opt.Value == nil {
			return profile, fmt.Errorf("smart charging key '%s' not found", chargingScheduleAllowedChargingRateUnit)
		}

		vals := strings.Split(*opt.Value, ",")
		profile.ChargingScheduleAllowedChargingRateUnit = append(profile.ChargingScheduleAllowedChargingRateUnit, vals...)
	}

	{ // optional
		var supported bool
		opt, found := options[string(connectorSwitch3to1PhaseSupported)]
		if found {
			supported, _ = strconv.ParseBool(*opt.Value)
		}

		profile.ConnectorSwitch3to1PhaseSupported = supported
	}

	return profile, nil

}

func parseIntOption(key smartchargingChargeProfileKey, options map[string]core.ConfigurationKey) (int, error) {
	opt, found := options[string(key)]
	if !found || opt.Value == nil {
		return 0, fmt.Errorf("smart charging key '%s' not found", key)
	}

	val, err := strconv.Atoi(*opt.Value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse key: %s", key)
	}

	return val, nil
}

// Boot waits for the CP to register itself
func (cp *CP) Boot() error {
	bootC := make(chan struct{})
	go func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		for cp.boot == nil || cp.status == nil {
			cp.initialized.Wait()
		}

		close(bootC)
	}()

	select {
	case <-bootC:
		cp.update()
		return nil
	case <-time.After(timeout):
		return api.ErrTimeout
	}
}

// TransactionID returns the current transaction id
func (cp *CP) TransactionID() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.txn
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.updated) > timeout {
		return res, api.ErrTimeout
	}

	if cp.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("chargepoint error: %s", cp.status.ErrorCode)
	}

	switch cp.status.Status {
	case core.ChargePointStatusAvailable, // "Available"
		core.ChargePointStatusUnavailable: // "Unavailable"
		res = api.StatusA
	case
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		res = api.StatusB
	case core.ChargePointStatusCharging: // "Charging"
		res = api.StatusC
	case core.ChargePointStatusReserved, // "Reserved"
		core.ChargePointStatusFaulted: // "Faulted"
		return api.StatusF, fmt.Errorf("chargepoint status: %s", cp.status.Status)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", cp.status.Status)
	}

	return res, nil
}
