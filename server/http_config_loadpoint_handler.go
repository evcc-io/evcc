package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/loadpoint"
	coresettings "github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
)

func getLoadpointStaticConfig(lp loadpoint.API) loadpoint.StaticConfig {
	return loadpoint.StaticConfig{
		Charger: lp.GetChargerRef(),
		Meter:   lp.GetMeterRef(),
		Circuit: lp.GetCircuitRef(),
		Vehicle: lp.GetDefaultVehicleRef(),
	}
}

func getLoadpointDynamicConfig(lp loadpoint.API) loadpoint.DynamicConfig {
	planTime, planEnergy := lp.GetPlanEnergy()
	return loadpoint.DynamicConfig{
		Title:                    lp.GetTitle(),
		DefaultMode:              string(lp.GetDefaultMode()),
		Priority:                 lp.GetPriority(),
		PhasesConfigured:         lp.GetPhasesConfigured(),
		MinCurrent:               lp.GetMinCurrent(),
		MaxCurrent:               lp.GetMaxCurrent(),
		SmartCostLimit:           lp.GetSmartCostLimit(),
		SmartFeedInPriorityLimit: lp.GetSmartFeedInPriorityLimit(),
		Thresholds:               lp.GetThresholds(),
		Soc:                      lp.GetSocConfig(),
		UI:                       lp.GetUI(),
		PlanEnergy:               planEnergy,
		PlanTime:                 planTime,
		PlanStrategy:             lp.GetPlanStrategy(),
		BatteryBoostLimit:        lp.GetBatteryBoostLimit(),
		LimitEnergy:              lp.GetLimitEnergy(),
		LimitSoc:                 lp.GetLimitSoc(),
	}
}

type loadpointFullConfig struct {
	ID      int    `json:"id,omitempty"` // db row id
	Name    string `json:"name"`         // either slice index (yaml) or db:<row id>
	Disable bool   `json:"disable,omitempty"`

	// static config
	loadpoint.StaticConfig
	loadpoint.DynamicConfig
}

func loadpointSplitConfig(r io.Reader) (loadpoint.DynamicConfig, map[string]any, bool, error) {
	var payload map[string]any

	if err := jsonDecoder(r).Decode(&payload); err != nil {
		return loadpoint.DynamicConfig{}, nil, false, err
	}

	disable, _ := payload["disable"].(bool)
	delete(payload, "disable")

	dynamic, static, err := loadpoint.SplitConfig(payload)
	return dynamic, static, disable, err
}

// loadpointConfig returns a single loadpoint's configuration
func loadpointConfig(dev config.Device[loadpoint.API]) (loadpointFullConfig, error) {
	var (
		id      int
		disable bool
	)
	if configurable, ok := dev.(config.ConfigurableDevice[loadpoint.API]); ok {
		id = configurable.ID()
		disable = configurable.Properties().Disable
	}

	lp := dev.Instance()

	// disabled loadpoint has no live instance; decode static config from the database instead
	if lp == nil {
		dynamic, staticMap, err := loadpoint.SplitConfig(dev.Config().Other)
		if err != nil {
			return loadpointFullConfig{}, err
		}

		var static loadpoint.StaticConfig
		if err := util.DecodeOther(staticMap, &static); err != nil {
			return loadpointFullConfig{}, err
		}

		return loadpointFullConfig{
			ID:            id,
			Name:          dev.Config().Name,
			Disable:       disable,
			StaticConfig:  static,
			DynamicConfig: dynamic,
		}, nil
	}

	res := loadpointFullConfig{
		ID:            id,
		Name:          dev.Config().Name,
		Disable:       disable,
		StaticConfig:  getLoadpointStaticConfig(lp),
		DynamicConfig: getLoadpointDynamicConfig(lp),
	}

	return res, nil
}

// loadpointsConfigHandler returns a device configurations by class
func loadpointsConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices := config.Loadpoints().Devices()

		res := make([]loadpointFullConfig, 0, len(devices))
		for _, dev := range devices {
			c, err := loadpointConfig(dev)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			res = append(res, c)
		}

		jsonWrite(w, res)
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

		res, err := loadpointConfig(dev)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, res)
	}
}

// newLoadpointHandler creates a new loadpoint
func newLoadpointHandler() http.HandlerFunc {
	h := config.Loadpoints()

	// TODO revert charger, meter etc

	return func(w http.ResponseWriter, r *http.Request) {
		dynamic, static, disable, err := loadpointSplitConfig(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		id := len(h.Devices())
		name := "lp-" + strconv.Itoa(id+1)
		log := util.NewLoggerWithLoadpoint(name, id+1)

		conf, err := config.AddConfig(templates.Loadpoint, static, config.WithProperties(config.Properties{Disable: disable}))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		settings := coresettings.NewConfigSettingsAdapter(log, &conf)

		instance, err := core.NewLoadpointFromConfig(log, settings, nil, static)
		if err != nil {
			conf.Delete()
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		dev := config.NewConfigurableDevice[loadpoint.API](&conf, instance)
		if err := dynamic.Apply(instance); err != nil {
			conf.Delete()
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := h.Add(dev); err != nil {
			conf.Delete()
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		setConfigDirty()

		w.WriteHeader(http.StatusOK)
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

		dynamic, static, disable, err := loadpointSplitConfig(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// static

		// merge here to maintain dynamic part of the config
		other := configurable.Config().Other
		if err := mergo.Merge(&other, static, mergo.WithOverride); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		instance := dev.Instance()

		if err := configurable.Update(other, instance, config.WithProperties(config.Properties{Disable: disable})); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// dynamic; instance is nil for a disabled loadpoint, takes effect on next restart
		if instance != nil {
			if err := dynamic.Apply(instance); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}

		setConfigDirty()

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

		// cleanup references
		lp, err := configurableDevice(config.NameForID(id), h)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		instance := lp.Instance()

		// disabled loadpoint has no live instance; delete it without ref cleanup
		if instance == nil {
			if err := deleteDevice(id, h); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			jsonWrite(w, struct {
				ID int `json:"id"`
			}{ID: id})

			return
		}

		if dev, err := configurableDevice(instance.GetChargerRef(), config.Chargers()); err == nil {
			if err := deleteDevice(dev.ID(), config.Chargers()); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			setConfigDirty()
		}

		if dev, err := configurableDevice(instance.GetMeterRef(), config.Meters()); err == nil {
			if err := deleteDevice(dev.ID(), config.Meters()); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			setConfigDirty()
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

		jsonWrite(w, res)
	}
}
