package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/loadpoint"
	coresettings "github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

type loadpointStaticConfig struct {
	// static config
	Charger string `json:"charger,omitempty"`
	Meter   string `json:"meter,omitempty"`
	Circuit string `json:"circuit,omitempty"`
	Vehicle string `json:"vehicle,omitempty"`
}

func getLoadpointStaticConfig(lp loadpoint.API) loadpointStaticConfig {
	return loadpointStaticConfig{
		Charger: lp.GetChargerName(),
		Meter:   lp.GetMeterName(),
		Circuit: lp.GetCircuitName(),
		Vehicle: lp.GetDefaultVehicle(),
	}
}

type loadpointDynamicConfig struct {
	// dynamic config
	Title          string   `json:"title"`
	Mode           string   `json:"mode"`
	Priority       int      `json:"priority"`
	Phases         int      `json:"phases"`
	MinCurrent     float64  `json:"minCurrent"`
	MaxCurrent     float64  `json:"maxCurrent"`
	SmartCostLimit *float64 `json:"smartCostLimit"`

	Thresholds loadpoint.ThresholdsConfig `json:"thresholds"`
	Soc        loadpoint.SocConfig        `json:"soc"`
}

func getLoadpointDynamicConfig(lp loadpoint.API) loadpointDynamicConfig {
	return loadpointDynamicConfig{
		Title:          lp.GetTitle(),
		Mode:           string(lp.GetMode()),
		Priority:       lp.GetPriority(),
		Phases:         lp.GetPhases(),
		MinCurrent:     lp.GetMinCurrent(),
		MaxCurrent:     lp.GetMaxCurrent(),
		SmartCostLimit: lp.GetSmartCostLimit(),
		Thresholds:     lp.GetThresholds(),
		Soc:            lp.GetSocConfig(),
	}
}

func loadpointUpdateDynamicConfig(payload loadpointDynamicConfig, lp loadpoint.API) error {
	lp.SetTitle(payload.Title)
	lp.SetPriority(payload.Priority)
	lp.SetSmartCostLimit(payload.SmartCostLimit)
	lp.SetThresholds(payload.Thresholds)

	// TODO mode warning
	lp.SetSocConfig(payload.Soc)

	mode, err := api.ChargeModeString(payload.Mode)
	if err == nil {
		lp.SetMode(mode)
	}

	if err == nil {
		err = lp.SetPhases(payload.Phases)
	}

	if err == nil {
		err = lp.SetMinCurrent(payload.MinCurrent)
	}

	if err == nil {
		err = lp.SetMaxCurrent(payload.MaxCurrent)
	}

	return err
}

type loadpointFullConfig struct {
	ID   int    `json:"id,omitempty"` // db row id
	Name string `json:"name"`         // either slice index (yaml) or db:<row id>

	// static config
	loadpointStaticConfig
	loadpointDynamicConfig
}

func loadpointSplitConfig(r io.Reader) (loadpointDynamicConfig, map[string]any, error) {
	var payload map[string]any

	if err := jsonDecoder(r).Decode(&payload); err != nil {
		return loadpointDynamicConfig{}, nil, err
	}

	// split static and dynamic config via mapstructure
	var cc struct {
		loadpointDynamicConfig `mapstructure:",squash"`
		Other                  map[string]any `mapstructure:",remain"`
	}

	if err := util.DecodeOther(payload, &cc); err != nil {
		return loadpointDynamicConfig{}, nil, err
	}

	return cc.loadpointDynamicConfig, cc.Other, nil
}

// loadpointConfig returns a single loadpoint's configuration
func loadpointConfig(dev config.Device[loadpoint.API]) loadpointFullConfig {
	lp := dev.Instance()

	var id int
	if configurable, ok := dev.(config.ConfigurableDevice[loadpoint.API]); ok {
		id = configurable.ID()
	}

	res := loadpointFullConfig{
		ID:   id,
		Name: dev.Config().Name,

		loadpointStaticConfig:  getLoadpointStaticConfig(lp),
		loadpointDynamicConfig: getLoadpointDynamicConfig(lp),
	}

	return res
}

// loadpointsConfigHandler returns a device configurations by class
func loadpointsConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := lo.Map(config.Loadpoints().Devices(), func(dev config.Device[loadpoint.API], _ int) loadpointFullConfig {
			return loadpointConfig(dev)
		})

		jsonResult(w, res)
	}
}

// loadpointConfigHandler returns a device configurations by class
func loadpointConfigHandler() http.HandlerFunc {
	h := config.Loadpoints()

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		dev, err := h.ByName(config.NameForID(id))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := loadpointConfig(dev)

		jsonResult(w, res)
	}
}

// newLoadpointHandler creates a new loadpoint
func newLoadpointHandler() http.HandlerFunc {
	h := config.Loadpoints()

	return func(w http.ResponseWriter, r *http.Request) {
		dynamic, static, err := loadpointSplitConfig(r.Body)

		id := len(h.Devices())
		name := "lp-" + strconv.Itoa(id+1)

		log := util.NewLoggerWithLoadpoint(name, id+1)

		dev := config.BlankConfigurableDevice[loadpoint.API]()
		settings := coresettings.NewDeviceSettingsAdapter(dev)

		instance, err := core.NewLoadpointFromConfig(log, settings, static)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		dev.Update(static, instance)

		if err := loadpointUpdateDynamicConfig(dynamic, instance); err != nil {
			jsonError(w, http.StatusBadRequest, err)
		}

		conf, err := config.AddConfig(templates.Loadpoint, "", static)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := h.Add(dev); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		setConfigDirty()

		res := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{
			ID:   conf.ID,
			Name: config.NameForID(conf.ID),
		}

		jsonResult(w, res)
	}
}

// updateLoadpointHandler returns a device configurations by class
func updateLoadpointHandler() http.HandlerFunc {
	h := config.Loadpoints()

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		dev, err := h.ByName(config.NameForID(id))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		configurable, ok := dev.(config.ConfigurableDevice[loadpoint.API])
		if !ok {
			jsonError(w, http.StatusBadRequest, errors.New("not configurable"))
			return
		}

		instance := dev.Instance()

		dynamic, static, err := loadpointSplitConfig(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// dynamic
		if err := loadpointUpdateDynamicConfig(dynamic, instance); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// static
		if err := configurable.Update(static, instance); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// TODO dirty handling
		w.WriteHeader(http.StatusOK)
	}
}

// deleteLoadpointHandler deletes a loadpoint
func deleteLoadpointHandler() http.HandlerFunc {
	h := config.Loadpoints()

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		setConfigDirty()

		if err := deleteDevice(id, h); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			ID int `json:"id"`
		}{
			ID: id,
		}

		jsonResult(w, res)
	}
}
