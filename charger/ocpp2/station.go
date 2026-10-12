package ocpp2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

type Station struct {
	mu  sync.RWMutex
	cs  *CS2
	log *util.Logger

	id        string
	connected bool

	onceConnect sync.Once
	connectC    chan struct{}

	bootTimer         *time.Timer
	bootNotificationC chan *provisioning.BootNotificationRequest
	BootNotification  *provisioning.BootNotificationResponse

	evses map[int]*EVSE
}

func NewStation(log *util.Logger, cs *CS2, id string) *Station {
	return &Station{
		cs:                cs,
		log:               log,
		id:                id,
		evses:             make(map[int]*EVSE),
		connectC:          make(chan struct{}),
		bootNotificationC: make(chan *provisioning.BootNotificationRequest, 1),
	}
}

func (st *Station) ID() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.id
}

func (st *Station) RegisterID(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.id != "" {
		panic("ocpp2: cannot re-register station id")
	}
	st.id = id
}

// ── EVSE-Verwaltung ───────────────────────────────────────────────────────────

func (st *Station) registerEVSE(id int, evse *EVSE) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.evses[id]; ok {
		return fmt.Errorf("evse already registered: %d", id)
	}
	st.evses[id] = evse
	return nil
}

func (st *Station) deregisterEVSE(id int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.evses, id)
}

func (st *Station) evseByID(id int) *EVSE {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.evses[id]
}

// ── Verbindungsstatus ─────────────────────────────────────────────────────────

func (st *Station) Connected() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.connected
}

func (st *Station) HasConnected() <-chan struct{} {
	return st.connectC
}

func (st *Station) connect(v bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.connected = v
	if v {
		st.onceConnect.Do(func() { close(st.connectC) })
	} else {
		st.stopBootTimer()
	}
}

func (st *Station) stopBootTimer() {
	if st.bootTimer != nil {
		st.bootTimer.Stop()
		st.bootTimer = nil
	}
}

func (st *Station) onTransportConnect() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.stopBootTimer()
	st.bootTimer = time.AfterFunc(Timeout, st.onBootTimeout)
}

func (st *Station) onBootTimeout() {
	st.mu.Lock()
	if st.bootTimer == nil {
		st.mu.Unlock()
		return
	}
	st.bootTimer = nil
	st.mu.Unlock()
	st.log.DEBUG.Printf("boot notification timeout – proceeding anyway")
	st.connect(true)
}

// ── Message Handler ───────────────────────────────────────────────────────────

func (st *Station) OnBootNotification(req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	st.mu.Lock()
	st.stopBootTimer()
	st.mu.Unlock()

	st.log.DEBUG.Printf("BootNotification: model=%s vendor=%s reason=%s",
		req.ChargingStation.Model, req.ChargingStation.VendorName, req.Reason)

	resp := &provisioning.BootNotificationResponse{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    int(heartbeatInterval.Seconds()),
		Status:      provisioning.RegistrationStatusAccepted,
	}
	st.BootNotification = resp

	select {
	case st.bootNotificationC <- req:
	default:
	}

	st.connect(true)
	return resp, nil
}

func (st *Station) OnStatusNotification(req *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	st.log.DEBUG.Printf("StatusNotification: evse=%d connector=%d status=%s",
		req.EvseID, req.ConnectorID, req.ConnectorStatus)
	if evse := st.evseByID(req.EvseID); evse != nil {
		return evse.OnStatusNotification(req)
	}
	return new(availability.StatusNotificationResponse), nil
}

func (st *Station) OnTransactionEvent(req *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	evseId := 0
	if req.Evse != nil {
		evseId = req.Evse.ID
	}
	st.log.DEBUG.Printf("TransactionEvent: evse=%d type=%s txn=%s",
		evseId, req.EventType, req.TransactionInfo.TransactionID)

	if evse := st.evseByID(evseId); evse != nil {
		return evse.OnTransactionEvent(req)
	}
	return &transactions.TransactionEventResponse{
		IDTokenInfo: &types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

func (st *Station) OnMeterValues(req *meter.MeterValuesRequest) (*meter.MeterValuesResponse, error) {
	if evse := st.evseByID(req.EvseID); evse != nil {
		return evse.OnMeterValues(req)
	}
	return new(meter.MeterValuesResponse), nil
}

func (st *Station) MonitorReboot(ctx context.Context, setup func() error) {
	select {
	case <-st.bootNotificationC:
	default:
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case boot := <-st.bootNotificationC:
				st.log.INFO.Printf("reboot detected (model=%s vendor=%s), re-initialising",
					boot.ChargingStation.Model, boot.ChargingStation.VendorName)
				if err := setup(); err != nil {
					st.log.ERROR.Printf("re-initialise after reboot failed: %v", err)
				}
			}
		}
	}()
}

// wait wartet auf Callback-Channel oder Timeout
func wait(err error, rc <-chan error) error {
	if err != nil {
		return err
	}
	select {
	case err = <-rc:
	case <-time.After(Timeout):
		err = api.ErrTimeout
	}
	return err
}