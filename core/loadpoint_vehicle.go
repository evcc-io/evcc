package core

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
)

const (
	vehicleDetectInterval = 1 * time.Minute
	vehicleDetectDuration = 10 * time.Minute
)

// availableVehicles is the slice of vehicles from the coordinator that are available
func (lp *Loadpoint) availableVehicles() []api.Vehicle {
	if lp.coordinator == nil {
		return nil
	}
	return lp.coordinator.GetVehicles(true)
}

// coordinatedVehicles is the slice of vehicles from the coordinator
func (lp *Loadpoint) coordinatedVehicles() []api.Vehicle {
	if lp.coordinator == nil {
		return nil
	}
	return lp.coordinator.GetVehicles(false)
}

// setVehicleIdentifier updated the vehicle id as read from the charger
func (lp *Loadpoint) setVehicleIdentifier(id string) {
	if lp.vehicleIdentifier != id {
		lp.vehicleIdentifier = id
		lp.publish(keys.VehicleIdentity, id)
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
func (lp *Loadpoint) setActiveVehicle(v api.Vehicle) {
	lp.vmu.Lock()

	from := "unknown"
	if lp.vehicle != nil {
		lp.coordinator.Release(lp.vehicle)
		from = lp.vehicle.GetTitle()
	}
	to := "unknown"
	if v != nil {
		lp.coordinator.Acquire(v)
		to = v.GetTitle()
	}

	lp.vehicle = v
	lp.vmu.Unlock()

	if from != to {
		lp.log.INFO.Printf("vehicle updated: %s -> %s", from, to)
	}

	if v != nil {
		lp.socUpdated = time.Time{}

		// resolve optional config
		var estimate bool
		if lp.Soc.Estimate == nil || *lp.Soc.Estimate {
			estimate = true
		}
		lp.socEstimator = soc.NewEstimator(lp.log, lp.charger, v, estimate)

		lp.publish(keys.VehicleName, vehicle.Settings(lp.log, v).Name())
		lp.publish(keys.VehicleTitle, v.GetTitle())

		if mode, ok := v.OnIdentified().GetMode(); ok {
			lp.SetMode(mode)
		}

		lp.addTask(lp.vehicleOdometer)

		lp.progress.Reset()
	} else {
		lp.socEstimator = nil
		lp.unpublishVehicleIdentity()
	}

	// re-publish vehicle settings
	lp.publish(keys.PhasesActive, lp.ActivePhases())
	lp.unpublishVehicle()

	// publish effective values
	lp.PublishEffectiveValues()

	lp.updateSession(func(session *session.Session) {
		var title string
		if v != nil {
			title = v.GetTitle()
		}

		lp.session.Vehicle = title
	})
}

func (lp *Loadpoint) wakeUpVehicle() {
	// wake up charger or vehicle. First wakeupAttemptsLeft will be odd.
	charger, chargerCanWakeUp := lp.charger.(api.Resurrector)
	vehicle, vehicleCanWakeUp := lp.GetVehicle().(api.Resurrector)

	if lp.wakeUpTimer.wakeupAttemptsLeft%2 != 0 {
		if chargerCanWakeUp {
			lp.wakeUpResurrector(charger, "charger")
		} else if vehicleCanWakeUp {
			lp.wakeUpResurrector(vehicle, "vehicle")
		}
	} else {
		if chargerCanWakeUp && vehicleCanWakeUp {
			lp.wakeUpResurrector(vehicle, "vehicle")
		}
	}
}

func (lp *Loadpoint) wakeUpResurrector(resurrector api.Resurrector, name string) {
	lp.log.DEBUG.Printf("wake-up %s, attempts left: %d", name, lp.wakeUpTimer.wakeupAttemptsLeft)
	if err := resurrector.WakeUp(); err != nil {
		lp.log.ERROR.Printf("wake-up %s: %v", name, err)
	}
}

// unpublishVehicleIdentity resets published vehicle identification
func (lp *Loadpoint) unpublishVehicleIdentity() {
	lp.publish(keys.VehicleName, "")
	lp.publish(keys.VehicleTitle, "")
}

// unpublishVehicle resets published vehicle data
func (lp *Loadpoint) unpublishVehicle() {
	lp.vehicleSoc = 0

	lp.publish(keys.VehicleClimaterActive, nil)
	lp.publish(keys.VehicleSoc, 0.0)
	lp.publish(keys.VehicleRange, int64(0))
	lp.publish(keys.VehicleLimitSoc, 0.0)
	lp.publish(keys.VehicleOdometer, 0.0)

	lp.setRemainingEnergy(0)
	lp.setRemainingDuration(0)
}

// vehicleHasFeature checks availability of vehicle feature
func (lp *Loadpoint) vehicleHasFeature(f api.Feature) bool {
	return hasFeature(lp.GetVehicle(), f)
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
		util.ResetCached()
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
	util.ResetCached()

	lp.vehicleDetect = lp.clock.Now()
	lp.vehicleDetectTicker = lp.clock.Ticker(vehicleDetectInterval)
	lp.publish(keys.VehicleDetectionActive, true)
}

// stopVehicleDetection expires the connection timer and ticker
func (lp *Loadpoint) stopVehicleDetection() {
	lp.vehicleDetect = time.Time{}
	if lp.vehicleDetectTicker != nil {
		lp.vehicleDetectTicker.Stop()
	}
	lp.publish(keys.VehicleDetectionActive, false)
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
			lp.publish(keys.VehicleOdometer, odo)

			// update session once odometer is read
			lp.updateSession(func(session *session.Session) {
				session.Odometer = &odo
			})
		} else if !loadpoint.AcceptableError(err) {
			lp.log.ERROR.Printf("vehicle odometer: %v", err)
		}
	}
}

// vehicleClimatePollAllowed determines if polling depending on mode and connection status
func (lp *Loadpoint) vehicleClimatePollAllowed() bool {
	if lp.charging() || lp.vehicleHasFeature(api.Streaming) {
		return true
	}

	return (lp.Soc.Poll.Mode == loadpoint.PollConnected || lp.Soc.Poll.Mode == loadpoint.PollAlways) && lp.connected()
}

// vehicleSocPollAllowed validates charging state against polling mode
func (lp *Loadpoint) vehicleSocPollAllowed() bool {
	// always update soc when charging
	if lp.charging() || lp.vehicleHasFeature(api.Streaming) {
		return true
	}

	// update if connected and soc unknown
	if lp.connected() && lp.socUpdated.IsZero() {
		return true
	}

	remaining := lp.Soc.Poll.Interval - lp.clock.Since(lp.socUpdated)

	honourUpdateInterval := lp.Soc.Poll.Mode == loadpoint.PollAlways ||
		lp.connected() && lp.Soc.Poll.Mode == loadpoint.PollConnected

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

			lp.publish(keys.VehicleClimaterActive, active)
			return active
		}

		if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("climater: %v", err)
		}
	}

	return false
}
