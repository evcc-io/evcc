package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// resolveBrand looks up a brand by name, case-insensitively
func resolveBrand(name string) (brand, bool) {
	for k, b := range brands {
		if strings.EqualFold(k, name) {
			return b, true
		}
	}
	return brand{}, false
}

// IsBrand reports whether name is a known VW group brand (case-insensitive)
func IsBrand(name string) bool {
	_, ok := resolveBrand(name)
	return ok
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

// dataPoint is a single data point as delivered in the dataset JSON document.
// Key is the data point's unique GUID, used when DataFieldName is generic (e.g. "value").
type dataPoint struct {
	Key           string     `json:"key"`
	DataFieldName string     `json:"dataFieldName"`
	Value         string     `json:"value"`
	TimestampUtc  *time.Time `json:"timestampUtc"`
}

// point is a decoded data point: its unique GUID (Key), delivered field Name,
// value, record time and dataset delivery sequence (higher Seq is newer).
type point struct {
	Key       string
	Name      string
	Value     string
	Timestamp time.Time
	Seq       uint64
}

// id is the point's deduplication and lookup identity: its unique GUID when
// present, otherwise the (possibly non-unique) field name.
func (p point) id() string {
	if p.Key != "" {
		return p.Key
	}
	return p.Name
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
	FieldChargingScenario             = "charging_state_report.charging_scenario"
	FieldPlugState                    = "plug_state"

	// soc
	FieldSoc                 = "state_of_charge"
	FieldHvSoc               = "hv_soc"
	FieldHvBatteryLevelValue = "battery_level_HV.value"

	// target soc
	FieldTargetSoc = "settings.target_soc"

	// range
	FieldRangeCombined       = "cruising_range_combined"
	FieldRangePrimary        = "cruising_range_primary_engine"
	FieldRangeSecondary      = "cruising_range_secondary_engine"
	KeyRangeID3              = "0ca40e18-0564-3eda-bcc0-7aee9ef44f04" // VW ID.3 range
	KeyBatteryStateReportSoc = "506cb83e-f99f-3af3-bbeb-0429b69a78d9" // VW ID.3 soc

	// odo
	FieldOdometer      = "mileage"
	FieldOdometerValue = "mileage.value"

	// time
	FieldRemainingTime = "remaining_charging_time"
)

// contentDatasets returns the content datasets, sorted oldest to newest. The
// portal emits "..._no_content_found.zip" placeholders while the vehicle is
// asleep; those are skipped.
func contentDatasets(list []dataset) []dataset {
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

	return content
}

// parseDataset extracts the inner JSON document from the dataset zip archive and
// decodes it into its data points.
func parseDataset(b []byte) ([]point, error) {
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

	var ds datasetFile
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}

	return points(ds.Data), nil
}

// points decodes data points, keeping the newest entry per id (see point.id).
func points(data []dataPoint) []point {
	var res []point

	for _, dp := range data {
		if dp.Value == "" {
			continue
		}

		var ts time.Time
		if dp.TimestampUtc != nil {
			ts = *dp.TimestampUtc
		}
		p := point{Key: dp.Key, Name: dp.DataFieldName, Value: dp.Value, Timestamp: ts}
		if p.id() == "" {
			continue
		}

		if e := find(res, p.id()); e != nil {
			// newest wins; on equal timestamps the later entry wins
			if !e.Timestamp.After(p.Timestamp) {
				*e = p
			}
			continue
		}
		res = append(res, p)
	}

	return res
}

// find returns the data point identified by id, matched by Key first and Name
// second, or nil if none is present.
func find(data []point, id string) *point {
	if i := slices.IndexFunc(data, func(p point) bool { return p.Key == id || p.Name == id }); i >= 0 {
		return &data[i]
	}
	return nil
}
