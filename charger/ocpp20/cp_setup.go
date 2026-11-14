package ocpp20

import (
	"errors"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// OCPP 2.0.1 standardized variable names used for capability discovery.
const (
	ComponentSmartChargingCtrlr = "SmartChargingCtrlr"
	VariableACPhaseSwitching    = "ACPhaseSwitchingSupported"
)

// Setup discovers charging station capabilities via GetVariables.
// Variables that aren't supported by the station are silently ignored;
// only successfully-read values update the Station state.
func (cp *Station) Setup() error {
	type result struct {
		resp *provisioning.GetVariablesResponse
		err  error
	}
	rc := make(chan result, 1)

	data := []provisioning.GetVariableData{
		{
			Component: types.Component{Name: ComponentSmartChargingCtrlr},
			Variable:  types.Variable{Name: VariableACPhaseSwitching},
		},
	}

	err := Instance().GetVariables(cp.ID(), func(resp *provisioning.GetVariablesResponse, err error) {
		rc <- result{resp, err}
	}, data)
	if err != nil {
		return err
	}

	select {
	case res := <-rc:
		if res.err != nil {
			return res.err
		}
		cp.applyVariables(res.resp.GetVariableResult)
		return nil
	case <-time.After(ocpp.Timeout):
		return errors.New("get variables: timeout")
	}
}

// applyVariables updates Station state from GetVariables results.
func (cp *Station) applyVariables(results []provisioning.GetVariableResult) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// cap to bound log volume from a misbehaving station
	const maxResults = 32
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	for _, r := range results {
		if r.AttributeStatus != provisioning.GetVariableStatusAccepted {
			cp.log.DEBUG.Printf("variable %s/%s: %s", r.Component.Name, r.Variable.Name, r.AttributeStatus)
			continue
		}

		switch r.Variable.Name {
		case VariableACPhaseSwitching:
			if v, err := strconv.ParseBool(r.AttributeValue); err == nil {
				cp.PhaseSwitching = v
				cp.log.DEBUG.Printf("phase switching supported: %t", v)
			}
		}
	}
}
