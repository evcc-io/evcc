package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"slices"
	"strings"
	"time"
)

// brand holds the OIDC client id and state suffix for a VW group brand.
// All brands share the same portal and endpoints and differ only in the
// identity client id (see lib/euDataAct.js BRAND_CLIENT_IDS).
type brand struct {
	clientID string
	state    string
}

// brands maps the configurable brand name to its identity parameters
var brands = map[string]brand{
	"Volkswagen": {"9b58543e-1c15-4193-91d5-8a14145bebb0@apps_vw-dilab_com", "VOLKSWAGEN_PASSENGER_CARS"},
	"Audi":       {"cc29b87a-5e9a-4362-aecf-5adea6b01bbb@apps_vw-dilab_com", "AUDI"},
	"Skoda":      {"3ea88bf9-1d4e-4a68-b3ad-4098c1f1d246@apps_vw-dilab_com", "SKODA"},
	"Seat":       {"f85e5b69-e3b2-43aa-9c0d-1b7d0e0b576f@apps_vw-dilab_com", "SEAT"},
	"Cupra":      {"f85e5b69-e3b2-43aa-9c0d-1b7d0e0b576f@apps_vw-dilab_com", "CUPRA"},
}

// Brands returns the supported brand names
func Brands() []string {
	return []string{"Volkswagen", "Audi", "Skoda", "Seat", "Cupra"}
}

// resolveBrand looks up a brand by name, case-insensitively
func resolveBrand(name string) (brand, bool) {
	for k, b := range brands {
		if strings.EqualFold(k, name) {
			return b, true
		}
	}
	return brand{}, false
}

// Vehicle is a single entry of the portal vehicle list. VIN and name carry
// several alternative field names depending on the response variant.
type Vehicle struct {
	VIN                         string `json:"vin"`
	VehicleIdentificationNumber string `json:"vehicleIdentificationNumber"`
	NickName                    string `json:"nickName"`
	VehicleNickname             string `json:"vehicleNickname"`
	Nickname                    string `json:"nickname"`
	ModelName                   string `json:"modelName"`
}

// Vin returns the vehicle identification number from whichever field is set
func (v Vehicle) Vin() string {
	if v.VIN != "" {
		return v.VIN
	}
	return v.VehicleIdentificationNumber
}

// Name returns the first non-empty display name
func (v Vehicle) Name() string {
	for _, s := range []string{v.NickName, v.VehicleNickname, v.Nickname, v.ModelName} {
		if s != "" {
			return s
		}
	}
	return ""
}

// dataset describes a single delivered dataset file. Timestamp is the parsed
// delivery time; it is populated by contentDatasets from the file name or the
// createdOn field.
type dataset struct {
	Name      string    `json:"name"`
	CreatedOn time.Time `json:"createdOn"`
}

// dataPoint is a single data point as delivered in the dataset JSON document
type dataPoint struct {
	DataFieldName string     `json:"dataFieldName"`
	Value         string     `json:"value"`
	TimestampUtc  *time.Time `json:"timestampUtc"`
}

// point is a decoded data point: its value and the time it was recorded
type point struct {
	Value     string
	Timestamp time.Time
}

// datasetFile is the JSON document contained in a dataset zip archive
type datasetFile struct {
	VIN  string      `json:"vin"`
	Data []dataPoint `json:"Data"`
}

// data field names as delivered in the dataset (see lib/euDataActDictionary.json)
const (
	// status
	FieldChargingState                = "charging_state"
	FieldChargingPlug1ConnectionState = "charging_plug1_connectionstate"
	FieldCurrentChargeState           = "charging_state_report.current_charge_state"
	FieldPlugState                    = "plug_state"

	// soc
	FieldBatteryStateReportSoc = "battery_state_report.soc"
	FieldSoc                   = "state_of_charge"
	FieldHvSoc                 = "hv_soc"
	FieldHvBatteryLevel        = "battery_level_HV.value"

	// target soc
	FieldTargetSoc = "settings.target_soc"

	// range
	FieldRangeCombined  = "cruising_range_combined"
	FieldRangePrimary   = "cruising_range_primary_engine"
	FieldRangeSecondary = "cruising_range_secondary_engine"

	// odo
	FieldOdometer      = "mileage"
	FieldOdometerValue = "mileage.value"

	// time
	FieldRemainingTime = "remaining_charging_time"
)

// contentDatasets returns the datasets that actually carry content, with their
// delivery time parsed into Timestamp and sorted from oldest to newest. The
// portal emits "..._no_content_found.zip" placeholders while the vehicle is
// asleep, which are skipped. An error is returned when a content dataset's
// timestamp cannot be parsed.
func contentDatasets(list []dataset) ([]dataset, error) {
	content := make([]dataset, 0, len(list))
	for _, d := range list {
		if strings.HasSuffix(strings.ToLower(d.Name), "_no_content_found.zip") {
			continue
		}

		content = append(content, d)
	}

	slices.SortStableFunc(content, func(a, b dataset) int {
		return a.CreatedOn.Compare(b.CreatedOn)
	})

	return content, nil
}

// parseDataset extracts the inner JSON document from the dataset zip archive and
// decodes it into the dataset's VIN and a map of data points keyed by the dotted
// data field name. On duplicate field names the entry with the newest timestamp
// wins. The VIN is returned so the caller can drop datasets that do not belong
// to the requested vehicle.
func parseDataset(log *log.Logger, b []byte) (map[string]point, error) {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}

	var file *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".json") {
			file = f
			break
		}
	}
	if file == nil {
		return nil, errors.New("no json document in dataset")
	}

	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	log.Println(raw)

	var ds datasetFile
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}

	res := make(map[string]point, len(ds.Data))
	for _, p := range ds.Data {
		if p.DataFieldName == "" || p.Value == "" {
			continue
		}

		var ts time.Time
		if p.TimestampUtc != nil {
			ts = *p.TimestampUtc
		}

		if cur, ok := res[p.DataFieldName]; ok && cur.Timestamp.After(ts) {
			continue
		}

		res[p.DataFieldName] = point{Value: p.Value, Timestamp: ts}
	}

	return res, nil
}
