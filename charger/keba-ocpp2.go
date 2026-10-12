package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	ocpp2pkg "github.com/evcc-io/evcc/charger/ocpp2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

type KebaOCPP2 struct {
	*embed
	implement.Caps
	log       *util.Logger
	cs        *ocpp2pkg.CS2
	st        *ocpp2pkg.Station
	evse      *ocpp2pkg.EVSE
	stationID string
	evseID    int
	idTag     string
	current   float64
	enabled   bool
}

func init() {
	registry.AddCtx("keba-ocpp2", NewKebaOCPP2FromConfig)
}

func NewKebaOCPP2FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed     `mapstructure:",squash"`
		StationID string
		EvseID    int
		IdTag     string
		Meter     bool
	}{
		EvseID: 1,
		Meter:  true,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.StationID == "" {
		return nil, fmt.Errorf("keba-ocpp2: stationid darf nicht leer sein")
	}
	return NewKebaOCPP2(ctx, cc.embed, cc.StationID, cc.EvseID, cc.IdTag, cc.Meter)
}

func NewKebaOCPP2(ctx context.Context, e embed, stationID string, evseID int, idTag string, hasMeter bool) (*KebaOCPP2, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("keba-ocpp2")

	cs, err := ocpp2pkg.Instance2()
	if err != nil {
		return nil, fmt.Errorf("ocpp2 not configured: %w", err)
	}

	wb := &KebaOCPP2{
		embed:     &e,
		Caps:      implement.New(),
		log:       log,
		cs:        cs,
		stationID: stationID,
		evseID:    evseID,
		idTag:     idTag,
		current:   6,
	}

	var st *ocpp2pkg.Station
	st, err = cs.RegisterStation(
		stationID,
		func() *ocpp2pkg.Station {
			return ocpp2pkg.NewStation(log, cs, stationID)
		},
		func(st *ocpp2pkg.Station) error {
			// ← NEU: Setup in Goroutine – blockiert evcc-Start NICHT!
			go func() {
				if err := wb.setup(ctx, st, hasMeter); err != nil {
					log.ERROR.Printf("setup failed: %v", err)
				}
			}()
			return nil // ← sofort zurückkehren
		},
	)
	if err != nil {
		return nil, err
	}
	wb.st = st

	return wb, nil
}

func (wb *KebaOCPP2) setup(ctx context.Context, st *ocpp2pkg.Station, hasMeter bool) error {
	wb.log.DEBUG.Printf("waiting for station %s to connect...", wb.stationID)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-st.HasConnected():
		wb.log.INFO.Printf("station %s connected", wb.stationID)
	}

	// wb.st früh setzen damit setTxDefaultProfile() es nutzen kann.
	// RegisterStation() setzt wb.st erst nach Rückkehr – zu spät!
	wb.st = st // ← NEU

	evse, err := ocpp2pkg.NewEVSE(ctx, wb.log, wb.evseID, st, wb.idTag, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create evse %d: %w", wb.evseID, err)
	}
	wb.evse = evse

	if err := evse.Initialized(); err != nil {
		return fmt.Errorf("evse %d not initialized: %w", wb.evseID, err)
	}
	wb.log.INFO.Printf("evse %d initialized", wb.evseID)

	if err := wb.setTxDefaultProfile(wb.current); err != nil {
		wb.log.WARN.Printf("failed to set initial TxDefaultProfile: %v", err)
	}

	if hasMeter {
		implement.Has(wb, implement.Meter(wb.evse.CurrentPower))
		implement.Has(wb, implement.MeterEnergy(wb.evse.TotalEnergy))
		implement.Has(wb, implement.PhaseCurrents(wb.evse.Currents))
	}
	implement.Has(wb, implement.CurrentGetter(wb.evse.GetMaxCurrent))
	implement.Has(wb, implement.Identifier(wb.identify))

	st.MonitorReboot(ctx, func() error {
		return wb.setup(ctx, st, hasMeter)
	})

	return nil
}

// ── api.Charger Interface ─────────────────────────────────────────────────────

func (wb *KebaOCPP2) Status() (api.ChargeStatus, error) {
	if wb.evse == nil {
		return api.StatusA, nil
	}

	s, err := wb.evse.Status()
	if err != nil {
		return api.StatusA, err
	}

	switch s {
	case availability.ConnectorStatusOccupied:
		txn, err := wb.evse.TransactionID()
		if err != nil {
			return api.StatusB, err
		}
		if txn != "" {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	default:
		// Available, Faulted, Unavailable, Reserved → StatusA
		return api.StatusA, nil
	}
}

func (wb *KebaOCPP2) Enabled() (bool, error) {
	if wb.evse == nil {
		return false, nil
	}
	txn, err := wb.evse.TransactionID()
	if err != nil {
		return false, err
	}
	// Wenn Transaktion läuft → enabled State synchronisieren
	active := txn != ""
	if active {
		wb.enabled = true // ← State synchronisieren
	}
	return active || wb.enabled, nil
}

func (wb *KebaOCPP2) Enable(enable bool) error {
	var amps float64
	if enable {
		amps = wb.current
	}
	if err := wb.setTxDefaultProfile(amps); err != nil {
		return err
	}
	wb.enabled = enable
	return nil
}

func (wb *KebaOCPP2) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*KebaOCPP2)(nil)

func (wb *KebaOCPP2) MaxCurrentMillis(current float64) error {
	wb.current = current
	enabled, _ := wb.Enabled() // ← neu: Enabled() statt wb.enabled
	if enabled {
		return wb.setTxDefaultProfile(current)
	}
	return nil
}

// ── Private Helpers ───────────────────────────────────────────────────────────

func (wb *KebaOCPP2) setTxDefaultProfile(amps float64) error {
	if wb.st == nil {
		return fmt.Errorf("station not connected")
	}

	schedule := types.ChargingSchedule{
		ID:               1,                             // ← ID nicht Id
		ChargingRateUnit: types.ChargingRateUnitAmperes, // ← Amperes mit s!
		ChargingSchedulePeriod: []types.ChargingSchedulePeriod{
			{StartPeriod: 0, Limit: amps},
		},
	}

	profile := types.ChargingProfile{
		ID:                     ocpp2pkg.DefaultChargingProfileId, // ← ocpp2pkg. prefix!
		StackLevel:             ocpp2pkg.DefaultStackLevel,        // ← ocpp2pkg. prefix!
		ChargingProfilePurpose: types.ChargingProfilePurposeTxDefaultProfile,
		ChargingProfileKind:    types.ChargingProfileKindAbsolute,
		ChargingSchedule:       []types.ChargingSchedule{schedule},
	}

	wb.log.DEBUG.Printf("SetChargingProfile: evse=%d amps=%.1f (TxDefault)", wb.evseID, amps)
	return wb.st.SetChargingProfileRequest(wb.evseID, profile)
}

func (wb *KebaOCPP2) identify() ([]string, error) {
	if wb.evse == nil {
		return nil, nil
	}
	token := wb.evse.IdToken()
	if token == "" {
		return nil, nil
	}
	return []string{token}, nil
}
