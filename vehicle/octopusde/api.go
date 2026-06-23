package octopusde

import (
	"context"
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
	octoDeGql "github.com/evcc-io/evcc/tariff/octopusde/graphql"
	"github.com/evcc-io/evcc/util"
)

// API is the Octopus Energy Germany Kraken client for vehicle data. It reuses the
// authenticated Kraken GraphQL client from the tariff implementation so the JWT
// token source and auth transport are not duplicated.
type API struct {
	*octoDeGql.OctopusDeGraphQLClient
}

// NewAPI creates a Kraken API client authenticated via the given credentials.
func NewAPI(log *util.Logger, email, password string) (*API, error) {
	// the account number is discovered on demand and not needed for the shared client
	client, err := octoDeGql.NewClient(log, email, password, "")
	if err != nil {
		return nil, err
	}
	return &API{OctopusDeGraphQLClient: client}, nil
}

// krakenAccounts lists the accounts accessible to the authenticated user.
type krakenAccounts struct {
	Viewer struct {
		Accounts []struct {
			Number string
		}
	}
}

// Account returns the configured account number, or the first account accessible
// to the authenticated user when none is configured.
func (v *API) Account(account string) (string, error) {
	if account != "" {
		return account, nil
	}

	accounts, err := v.Accounts()
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", api.ErrNotAvailable
	}

	return accounts[0], nil
}

// Accounts returns the account numbers accessible to the authenticated user.
func (v *API) Accounts() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var q krakenAccounts
	if err := v.Query(ctx, &q, nil); err != nil {
		return nil, err
	}

	res := make([]string, 0, len(q.Viewer.Accounts))
	for _, a := range q.Viewer.Accounts {
		res = append(res, a.Number)
	}
	return res, nil
}

// socStatus holds the live state-of-charge values reported for a SmartFlex device.
// json.Number decodes both bare and quoted numbers (Kraken serialises soc as "41.00").
type socStatus struct {
	StateOfCharge struct {
		Value *json.Number
	}
	StateOfChargeLimit struct {
		UpperSocLimit *json.Number
	}
}

// Device is a SmartFlex device (vehicle or charge point) with its live status.
type Device struct {
	ID         string
	Name       string
	DeviceType string
	Provider   string
	Status     struct {
		// Both fragments select the same fields; the API returns whichever matches
		// the device's concrete type, so reading either yields the live values.
		Vehicle     socStatus `graphql:"... on SmartFlexVehicleStatus"`
		ChargePoint socStatus `graphql:"... on SmartFlexChargePointStatus"`
	}
}

// soc returns the populated state-of-charge status for the device.
func (d Device) soc() socStatus {
	if d.Status.Vehicle.StateOfCharge.Value != nil || d.Status.Vehicle.StateOfChargeLimit.UpperSocLimit != nil {
		return d.Status.Vehicle
	}
	return d.Status.ChargePoint
}

// parseNumber returns the float value of a reported json.Number, if present and valid.
func parseNumber(n *json.Number) (float64, bool) {
	if n == nil {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil
}

// Soc returns the battery state of charge in percent, if reported.
func (d Device) Soc() (float64, bool) {
	return parseNumber(d.soc().StateOfCharge.Value)
}

// TargetSoc returns the configured charge limit in percent, if any.
func (d Device) TargetSoc() (float64, bool) {
	return parseNumber(d.soc().StateOfChargeLimit.UpperSocLimit)
}

// krakenDevices lists the SmartFlex devices of an account.
type krakenDevices struct {
	Devices []Device `graphql:"devices(accountNumber: $accountNumber)"`
}

// Devices lists the SmartFlex devices of the given account.
func (v *API) Devices(accountNumber string) ([]Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var q krakenDevices
	if err := v.Query(ctx, &q, map[string]any{
		"accountNumber": accountNumber,
	}); err != nil {
		return nil, err
	}
	return q.Devices, nil
}
