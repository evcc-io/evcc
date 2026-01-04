// Package emhcasa provides EMH CASA 1.1 Smart Meter Gateway support.
// Based on work by gosanman (https://github.com/gosanman/smartmetergateway)
//
// MIT License
//
// Copyright (c) 2026 gosanman
// Copyright (c) 2026 iseeberg79
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package meter

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// hostHeaderTransport wraps a RoundTripper and enforces a custom Host header
type hostHeaderTransport struct {
	base http.RoundTripper
	host string
}

func (t *hostHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Host = t.host
	req.Header.Set("Host", t.host)
	return t.base.RoundTrip(req)
}

// parseURIHost extracts the host from a URI (without port/path)
func parseURIHost(uri string) (string, error) {
	uri = strings.TrimPrefix(uri, "https://")
	uri = strings.TrimPrefix(uri, "http://")

	if idx := strings.Index(uri, "/"); idx != -1 {
		uri = uri[:idx]
	}
	if idx := strings.Index(uri, ":"); idx != -1 {
		uri = uri[:idx]
	}

	if uri == "" {
		return "", fmt.Errorf("invalid uri")
	}

	return uri, nil
}

func init() {
	registry.Add("emh-casa", NewEMHCasaFromConfig)
}

// EMHCasa meter implementation for EMH CASA 1.1 Smart Meter Gateways
type EMHCasa struct {
	*request.Helper
	uri     string
	meterID string
	valuesG func() (map[string]float64, error)
	log     *util.Logger
}

// NewEMHCasaFromConfig creates meter from evcc config
func NewEMHCasaFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		MeterID  string
		Host     string // REQUIRED for most CASA gateways (example: 192.168.33.2)
		Refresh  time.Duration
	}{
		Refresh: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEMHCasa(cc.URI, cc.User, cc.Password, cc.MeterID, cc.Host, cc.Refresh)
}

// NewEMHCasa creates an EMH CASA meter
func NewEMHCasa(uri, user, password, meterID, host string, refresh time.Duration) (api.Meter, error) {
	log := util.NewLogger("emh-casa").Redact(user, password)

	if host == "" {
		derived, err := parseURIHost(uri)
		if err != nil {
			return nil, fmt.Errorf("host required and could not be derived: %w", err)
		}

		log.WARN.Printf(
			"no host configured, using host derived from uri (%s). "+
				"EMH CASA usually requires the gateway IP (e.g. 192.168.33.2)",
			derived,
		)

		host = derived
	} else {
		log.DEBUG.Printf("using configured host header: %s", host)
	}

	// CASA gateways require HTTP/1.1 and usually use self-signed certs
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ForceAttemptHTTP2: false,
	}

	hostTransport := &hostHeaderTransport{
		base: customTransport,
		host: host,
	}

	helper := request.NewHelper(log)
	helper.Client.Transport = digest.NewTransport(user, password, hostTransport)

	m := &EMHCasa{
		Helper:  helper,
		uri:     util.DefaultScheme(uri, "https"),
		meterID: meterID,
		log:     log,
	}

	if m.meterID == "" {
		if err := m.discoverMeterID(); err != nil {
			return nil, fmt.Errorf("failed to discover meter ID")
		}
		prefix := m.meterID
		if len(prefix) > 4 {
			prefix = prefix[:4] + "..."
		}
		log.DEBUG.Printf("discovered meter ID: %s", prefix)
	} else {
		prefix := m.meterID
		if len(prefix) > 4 {
			prefix = prefix[:4] + "..."
		}
		log.DEBUG.Printf("using configured meter ID: %s", prefix)
	}

	// Redact meter ID for any future logs
	if m.meterID != "" {
		log.Redact(m.meterID)
	}

	// Validate connection
	log.DEBUG.Printf("validating connection...")
	if _, err := m.getMeterValues(); err != nil {
		return nil, fmt.Errorf("failed to validate meter connection: %w", err)
	}
	log.DEBUG.Printf("connection validated successfully")

	m.valuesG = util.Cached(m.getMeterValues, refresh)

	return m, nil
}

type derivedContract struct {
	TafType       string   `json:"taf_type"`
	SensorDomains []string `json:"sensor_domains"`
}

type meterValue struct {
	Value       string `json:"value"`
	Unit        int    `json:"unit"`   // 27 = W, 30 = Wh
	Scaler      int    `json:"scaler"` // power-of-10 multiplier
	LogicalName string `json:"logical_name"`
}

type meterReading struct {
	Values []meterValue `json:"values"`
}

// discoverMeterID finds the first TAF-1 contract
func (m *EMHCasa) discoverMeterID() error {
	var contracts []string
	uri := fmt.Sprintf("%s/json/metering/derived", m.uri)

	m.log.DEBUG.Printf("discovering meter ID")

	if err := m.GetJSON(uri, &contracts); err != nil {
		return fmt.Errorf("failed to get contracts")
	}

	m.log.DEBUG.Printf("found %d contract(s)", len(contracts))

	for _, id := range contracts {
		m.log.DEBUG.Printf("checking contract")

		var c derivedContract
		uri := fmt.Sprintf("%s/json/metering/derived/%s", m.uri, id)

		if err := m.GetJSON(uri, &c); err != nil {
			m.log.DEBUG.Printf("failed to get contract details")
			continue
		}

		m.log.DEBUG.Printf("checking contract with taf_type=%s", c.TafType)

		if c.TafType == "TAF-1" && len(c.SensorDomains) > 0 {
			m.meterID = c.SensorDomains[0]
			return nil
		}
	}

	return fmt.Errorf("no TAF-1 contract found among %d contracts", len(contracts))
}

// convertToOBIS converts CASA logical name to OBIS C.D.E
func convertToOBIS(logicalName string) (string, error) {
	hex := strings.SplitN(logicalName, ".", 2)[0]

	if len(hex) != 12 {
		return "", fmt.Errorf("unexpected logical name: %s", logicalName)
	}

	c, err := strconv.ParseInt(hex[4:6], 16, 64)
	if err != nil {
		return "", err
	}
	d, err := strconv.ParseInt(hex[6:8], 16, 64)
	if err != nil {
		return "", err
	}
	e, err := strconv.ParseInt(hex[8:10], 16, 64)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d.%d", c, d, e), nil
}

// getMeterValues fetches and parses meter readings
func (m *EMHCasa) getMeterValues() (map[string]float64, error) {
	var reading meterReading
	uri := fmt.Sprintf("%s/json/metering/origin/%s/extended", m.uri, m.meterID)

	if err := m.GetJSON(uri, &reading); err != nil {
		return nil, err
	}

	values := make(map[string]float64)

	for _, item := range reading.Values {
		obis, err := convertToOBIS(item.LogicalName)
		if err != nil {
			continue
		}

		raw, err := strconv.ParseFloat(item.Value, 64)
		if err != nil {
			continue
		}

		val := raw * math.Pow(10, float64(item.Scaler))

		switch item.Unit {
		case 27: // W (Watt)
			values[obis] = val
		case 30: // Wh (Watthour) → kWh
			values[obis] = val / 1000
		case 33: // A (Ampere)
			values[obis] = val
		case 35: // V (Volt)
			values[obis] = val
		case 44: // Hz (Hertz)
			values[obis] = val
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no valid meter values found")
	}

	return values, nil
}

// CurrentPower implements api.Meter (OBIS 16.7.0)
func (m *EMHCasa) CurrentPower() (float64, error) {
	return m.getOBISValue("16.7.0")
}

// TotalEnergy implements api.MeterEnergy (OBIS 1.8.0)
func (m *EMHCasa) TotalEnergy() (float64, error) {
	return m.getOBISValue("1.8.0")
}

// getOBISValue is a DRY helper for OBIS value extraction
func (m *EMHCasa) getOBISValue(obis string) (float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, err
	}

	v, ok := values[obis]
	if !ok {
		return 0, fmt.Errorf("OBIS value (%s) not found", obis)
	}

	return v, nil
}

// Currents implements api.PhaseCurrents
func (m *EMHCasa) Currents() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["31.7.0"]
	l2 := values["51.7.0"]
	l3 := values["71.7.0"]

	return l1, l2, l3, nil
}

// Voltages implements api.PhaseVoltages
func (m *EMHCasa) Voltages() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["32.7.0"]
	l2 := values["52.7.0"]
	l3 := values["72.7.0"]

	return l1, l2, l3, nil
}

// Powers implements api.PhasePowers
func (m *EMHCasa) Powers() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["36.7.0"]
	l2 := values["56.7.0"]
	l3 := values["76.7.0"]

	return l1, l2, l3, nil
}

// GridProduction returns grid feed-in energy (OBIS 2.8.0)
func (m *EMHCasa) GridProduction() (float64, error) {
	return m.getOBISValue("2.8.0")
}

var _ api.Meter = (*EMHCasa)(nil)
var _ api.MeterEnergy = (*EMHCasa)(nil)
var _ api.PhaseCurrents = (*EMHCasa)(nil)
var _ api.PhaseVoltages = (*EMHCasa)(nil)
var _ api.PhasePowers = (*EMHCasa)(nil)
