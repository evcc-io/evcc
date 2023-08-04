package core

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/db/settings"
)

const (
	vehicleDetectInterval = 1 * time.Minute
	vehicleDetectDuration = 10 * time.Minute
)

// coordinatedVehicles is the slice of vehicles from the coordinator
func (lp *Loadpoint) coordinatedVehicles() []api.Vehicle {
	if lp.coordinator == nil {
		return nil
	}
	return lp.coordinator.GetVehicles()
}

// setVehicleIdentifier updated the vehicle id as read from the charger
func (lp *Loadpoint) setVehicleIdentifier(id string) {
	if lp.vehicleIdentifier != id {
		lp.vehicleIdentifier = id
		lp.publish("vehicleIdentity", id)
	}
}

// identifyVehicle reads vehicle identification from charger
func (lp *Loadpoint) identifyVehicle() {
	identifier, ok := lp.charger.(api.Identifier)
	if !ok {
		return
	}

	id, err := identifier.Identify()
	if err != nil {
		lp.log.ERROR.Println("charger vehicle id:", err)
		return
	}

	if lp.vehicleIdentifier == id {
		return
	}

	// vehicle found or removed
	lp.setVehicleIdentifier(id)

	if id != "" {
		lp.log.DEBUG.Println("charger vehicle id:", id)

		if vehicle := lp.selectVehicleByID(id); vehicle != nil {
			lp.stopVehicleDetection()
			lp.setActiveVehicle(vehicle)
		}
	}
}

// selectVehicleByID selects the vehicle with the given ID
func (lp *Loadpoint) selectVehicleByID(id string) api.Vehicle {
	vehicles := lp.coordinatedVehicles()

	// find exact match
	for _, vehicle := range vehicles {
		for _, vid := range vehicle.Identifiers() {
			if strings.EqualFold(id, vid) {
				return vehicle
			}
		}
	}

	// find placeholder match
	for _, vehicle := range vehicles {
		for _, vid := range vehicle.Identifiers() {
			// case insensitive match
			re, err := regexp.Compile("(?i)" + strings.ReplaceAll(vid, "*", ".*?"))
			if err != nil {
				lp.log.ERROR.Printf("vehicle id: %v", err)
				continue
			}

			if re.MatchString(id) {
				return vehicle
			}
		}
	}

	return nil
}

// setActiveVehicle assigns currently active vehicle, configures soc estimator
// and adds an odometer task
func (lp *Loadpoint) setActiveVehicle(vehicle api.Vehicle) {
	lp.vehicleMux.Lock()
	if lp.vehicle == vehicle {
		lp.vehicleMux.Unlock()
		return
	}

	from := "unknown"
	if lp.vehicle != nil {
		lp.coordinator.Release(lp.vehicle)
		from = lp.vehicle.Title()
	}
	to := "unknown"
	if vehicle != nil {
		lp.coordinator.Acquire(vehicle)
		to = vehicle.Title()
	}

	lp.vehicle = vehicle
	lp.vehicleMux.Unlock()

	lp.log.INFO.Printf("vehicle updated: %s -> %s", from, to)

	// lock api
	lp.Lock()

	// reset minSoc and targetSoc before change
	lp.setMinSoc(0)
	lp.setTargetSoc(100)

	// reset target energy
	lp.setTargetEnergy(0)

	// unlock api
	lp.Unlock()

	if vehicle != nil {
		lp.socUpdated = time.Time{}

		// resolve optional config
		var estimate bool
		if lp.Soc.Estimate == nil || *lp.Soc.Estimate {
			estimate = true
		}
		lp.socEstimator = soc.NewEstimator(lp.log, lp.charger, vehicle, estimate)

		lp.publish(vehiclePresent, true)
		lp.publish(vehicleTitle, vehicle.Title())
		lp.publish(vehicleIcon, vehicle.Icon())
		lp.publish(vehicleCapacity, vehicle.Capacity())
		lp.restoreVehicleSettings()

		lp.applyAction(vehicle.OnIdentified())
		lp.addTask(lp.vehicleOdometer)

		lp.progress.Reset()
	} else {
		lp.socEstimator = nil

		lp.publish(vehiclePresent, false)
		lp.publish(vehicleTitle, "")
		lp.publish(vehicleIcon, "")
		lp.publish(vehicleCapacity, int64(0))
		lp.publish(vehicleOdometer, 0.0)
	}

	// re-publish vehicle settings
	lp.publish(phasesActive, lp.activePhases())
	lp.unpublishVehicle()

	lp.updateSession(func(session *session.Session) {
		var title string
		if vehicle != nil {
			title = vehicle.Title()
		}

		lp.session.Vehicle = title
	})
}

func (lp *Loadpoint) wakeUpVehicle() {
	// charger
	if c, ok := lp.charger.(api.Resurrector); ok {
		lp.log.DEBUG.Println("wake-up charger")
		if err := c.WakeUp(); err != nil {
			lp.log.ERROR.Printf("wake-up charger: %v", err)
		}
	}

	// vehicle
	if vs, ok := lp.GetVehicle().(api.Resurrector); ok {
		lp.log.DEBUG.Println("wake-up vehicle")
		if err := vs.WakeUp(); err != nil {
			lp.log.ERROR.Printf("wake-up vehicle: %v", err)
		}
	}
}

// unpublishVehicle resets published vehicle data
func (lp *Loadpoint) unpublishVehicle() {
	lp.vehicleSoc = 0

	lp.publish(vehicleSoc, 0.0)
	lp.publish(vehicleRange, int64(0))
	lp.publish(vehicleTargetSoc, 0.0)

	lp.setRemainingEnergy(0)
	lp.setRemainingDuration(0)

	lp.publishVehicleFeature(api.Offline)
}

// vehicleHasFeature checks availability of vehicle feature
func (lp *Loadpoint) vehicleHasFeature(f api.Feature) bool {
	v, ok := lp.GetVehicle().(api.FeatureDescriber)
	if ok {
		ok = slices.Contains(v.Features(), f)
	}
	return ok
}

// publishVehicleFeature availability of vehicle features
func (lp *Loadpoint) publishVehicleFeature(f api.Feature) {
	lp.publish("vehicleFeature"+f.String(), lp.vehicleHasFeature(f))
}

// persistVehicleSettings stores user configuration (via UI/API) for the current vehicle
func (lp *Loadpoint) persistVehicleSettings() {
	idx := lp.coordinator.GetVehicleIndex(lp.GetVehicle())
	if idx == -1 {
		return
	}
	settings.SetInt(fmt.Sprintf("vehicle.%d.targetSoc", idx), int64(lp.Soc.target))
	settings.SetFloat(fmt.Sprintf("vehicle.%d.targetEnergy", idx), lp.targetEnergy)
	settings.SetInt(fmt.Sprintf("vehicle.%d.minSoc", idx), int64(lp.Soc.min))
	settings.SetTime(fmt.Sprintf("vehicle.%d.targetTime", idx), lp.targetTime)
}

// restoreVehicleSettings restores user configuration (via UI/API) for the current vehicle
func (lp *Loadpoint) restoreVehicleSettings() {
	idx := lp.coordinator.GetVehicleIndex(lp.GetVehicle())
	if idx == -1 {
		return
	}
	if v, err := settings.Int(fmt.Sprintf("vehicle.%d.targetSoc", idx)); err == nil {
		lp.setTargetSoc(int(v))
	}
	if v, err := settings.Float(fmt.Sprintf("vehicle.%d.targetEnergy", idx)); err == nil {
		lp.setTargetEnergy(v)
	}
	if v, err := settings.Int(fmt.Sprintf("vehicle.%d.minSoc", idx)); err == nil {
		lp.setMinSoc(int(v))
	}
	if v, err := settings.Time(fmt.Sprintf("vehicle.%d.targetTime", idx)); err == nil {
		if v.After(time.Now()) {
			lp.setTargetTime(v)
		}
	}
}

// vehicleUnidentified returns true if there are associated vehicles and detection is running.
// It will also reset the api cache at regular intervals.
// Detection is stopped after maximum duration and the "guest vehicle" message dispatched.
func (lp *Loadpoint) vehicleUnidentified() bool {
	if lp.vehicle != nil || lp.vehicleDetect.IsZero() || len(lp.coordinatedVehicles()) == 0 {
		return false
	}

	// stop detection
	if lp.clock.Since(lp.vehicleDetect) > vehicleDetectDuration {
		lp.stopVehicleDetection()
		lp.pushEvent(evVehicleUnidentified)
		return false
	}

	// request vehicle api refresh while waiting to identify
	select {
	case <-lp.vehicleDetectTicker.C:
		lp.log.DEBUG.Println("vehicle api refresh")
		provider.ResetCached()
	default:
	}

	return true
}

// vehicleDefaultOrDetect will assign and update default vehicle or start detection
func (lp *Loadpoint) vehicleDefaultOrDetect() {
	if lp.defaultVehicle != nil {
		if lp.vehicle != lp.defaultVehicle {
			lp.setActiveVehicle(lp.defaultVehicle)
		} else {
			// default vehicle is already active, update odometer anyway
			// need to do this here since setActiveVehicle would short-circuit
			lp.addTask(lp.vehicleOdometer)
		}
	} else if len(lp.coordinatedVehicles()) > 0 && lp.connected() {
		lp.startVehicleDetection()
	}
}

// startVehicleDetection reset connection timer and starts api refresh timer
func (lp *Loadpoint) startVehicleDetection() {
	// flush all vehicles before detection starts
	lp.log.DEBUG.Println("vehicle api refresh")
	provider.ResetCached()

	lp.vehicleDetect = lp.clock.Now()
	lp.vehicleDetectTicker = lp.clock.Ticker(vehicleDetectInterval)
	lp.publish(vehicleDetectionActive, true)
}

// stopVehicleDetection expires the connection timer and ticker
func (lp *Loadpoint) stopVehicleDetection() {
	lp.vehicleDetect = time.Time{}
	if lp.vehicleDetectTicker != nil {
		lp.vehicleDetectTicker.Stop()
	}
	lp.publish(vehicleDetectionActive, false)
}

// identifyVehicleByStatus validates if the active vehicle is still connected to the loadpoint
func (lp *Loadpoint) identifyVehicleByStatus() {
	if len(lp.coordinatedVehicles()) == 0 {
		return
	}

	if vehicle := lp.coordinator.IdentifyVehicleByStatus(); vehicle != nil {
		lp.stopVehicleDetection()
		lp.setActiveVehicle(vehicle)
		return
	}

	// remove previous vehicle if status was not confirmed
	if _, ok := lp.GetVehicle().(api.ChargeState); ok {
		lp.setActiveVehicle(nil)
	}
}

// vehicleOdometer updates odometer
func (lp *Loadpoint) vehicleOdometer() {
	if vs, ok := lp.GetVehicle().(api.VehicleOdometer); ok {
		if odo, err := vs.Odometer(); err == nil {
			lp.log.DEBUG.Printf("vehicle odometer: %.0fkm", odo)
			lp.publish(vehicleOdometer, odo)

			// update session once odometer is read
			lp.updateSession(func(session *session.Session) {
				session.Odometer = &odo
			})
		} else if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("vehicle odometer: %v", err)
		}
	}
}

// vehicleClimatePollAllowed determines if polling depending on mode and connection status
func (lp *Loadpoint) vehicleClimatePollAllowed() bool {
	switch {
	case lp.Soc.Poll.Mode == pollCharging && lp.charging():
		return true
	case (lp.Soc.Poll.Mode == pollConnected || lp.Soc.Poll.Mode == pollAlways) && lp.connected():
		return true
	default:
		return false
	}
}

// vehicleSocPollAllowed validates charging state against polling mode
func (lp *Loadpoint) vehicleSocPollAllowed() bool {
	// always update soc when charging
	if lp.charging() {
		return true
	}

	// update if connected and soc unknown
	if lp.connected() && lp.socUpdated.IsZero() {
		return true
	}

	remaining := lp.Soc.Poll.Interval - lp.clock.Since(lp.socUpdated)

	honourUpdateInterval := lp.Soc.Poll.Mode == pollAlways ||
		lp.connected() && lp.Soc.Poll.Mode == pollConnected

	if honourUpdateInterval {
		if remaining > 0 {
			lp.log.DEBUG.Printf("next soc poll remaining time: %v", remaining.Truncate(time.Second))
		} else {
			return true
		}
	}

	return false
}

// vehicleClimateActive checks if vehicle has active climate request
func (lp *Loadpoint) vehicleClimateActive() bool {
	if cl, ok := lp.GetVehicle().(api.VehicleClimater); ok && lp.vehicleClimatePollAllowed() {
		active, err := cl.Climater()
		if err == nil {
			if active {
				lp.log.DEBUG.Println("climater active")
			}

			lp.publish("climaterActive", active)
			return active
		}

		if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("climater: %v", err)
		}
	}

	return false
}
