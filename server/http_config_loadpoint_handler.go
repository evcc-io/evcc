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
	"github.com/samber/lo"
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
	planTime, planPrecondition, planEnergy := lp.GetPlanEnergy()
	return loadpoint.DynamicConfig{
		Title:            lp.GetTitle(),
		DefaultMode:      string(lp.GetDefaultMode()),
		Priority:         lp.GetPriority(),
		PhasesConfigured: lp.GetPhasesConfigured(),
		MinCurrent:       lp.GetMinCurrent(),
		MaxCurrent:       lp.GetMaxCurrent(),
		SmartCostLimit:   lp.GetSmartCostLimit(),
		Thresholds:       lp.GetThresholds(),
		Soc:              lp.GetSocConfig(),
		PlanEnergy:       planEnergy,
		PlanTime:         planTime,
		PlanPrecondition: int64(planPrecondition.Seconds()),
		LimitEnergy:      lp.GetLimitEnergy(),
		LimitSoc:         lp.GetLimitSoc(),
	}
}

type loadpointFullConfig struct {
	ID   int    `json:"id,omitempty"` // db row id
	Name string `json:"name"`         // either slice index (yaml) or db:<row id>

	// static config
	loadpoint.StaticConfig
	loadpoint.DynamicConfig
}

func loadpointSplitConfig(r io.Reader) (loadpoint.DynamicConfig, map[string]any, error) {
	var payload map[string]any

	if err := jsonDecoder(r).Decode(&payload); err != nil {
		return loadpoint.DynamicConfig{}, nil, err
	}

	return loadpoint.SplitConfig(payload)
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

		StaticConfig:  getLoadpointStaticConfig(lp),
		DynamicConfig: getLoadpointDynamicConfig(lp),
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

	// TODO revert charger, meter etc

	return func(w http.ResponseWriter, r *http.Request) {
		dynamic, static, err := loadpointSplitConfig(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		id := len(h.Devices())
		name := "lp-" + strconv.Itoa(id+1)
		log := util.NewLoggerWithLoadpoint(name, id+1)

		conf, err := config.AddConfig(templates.Loadpoint, static)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		settings := coresettings.NewConfigSettingsAdapter(log, &conf)
		instance, err := core.NewLoadpointFromConfig(log, settings, static)
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

		dynamic, static, err := loadpointSplitConfig(r.Body)
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

		if err := configurable.Update(other, instance); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// dynamic
		if err := dynamic.Apply(instance); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
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

		jsonResult(w, res)
	}
}
