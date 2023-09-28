package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"

	//"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// TODO support multiple connectors
// Since ocpp-go interfaces at charge point level, we need to manage multiple connector separately

// TransactionInfo contains info about a transaction
type TransactionInfo struct {
	id          int
	startTime   *types.DateTime
	endTime     *types.DateTime
	startMeter  int
	endMeter    int
	connectorId int
	idTag       string
}

func (ti *TransactionInfo) hasTransactionEnded() bool {
	return ti.endTime != nil && !ti.endTime.IsZero()
}

type ConnectorInfo struct {
	status             *core.StatusNotificationRequest
	availability       core.AvailabilityType
	currentTransaction int
	currentReservation int

	measurements map[string]types.SampledValue
	meterUpdated time.Time
	statusC      chan struct{}
	txnId        int
}

type CP struct {
	clock clock.Clock // mockable time
	mu    sync.Mutex
	once  sync.Once
	log   *util.Logger

	id           string
	connectors   map[int]*ConnectorInfo
	transactions map[int]*TransactionInfo

	txnCount int // change initial value to the last known global transaction. Needs persistence

	connectC  chan struct{}
	connected bool

	timeout time.Duration
	//status            *core.StatusNotificationRequest
}

func NewChargePoint(log *util.Logger, id string, connector int, timeout time.Duration) *CP {

	//ocpp.Instance()

	newCP := &CP{
		clock:        clock.New(),
		log:          log,
		id:           id,
		connectors:   make(map[int]*ConnectorInfo),
		transactions: make(map[int]*TransactionInfo),
		connectC:     make(chan struct{}),
		timeout:      timeout,

		//measurements: make(map[string]types.SampledValue),

	}

	//newCP.initialiseDefaultConnector(connector, timeout)

	return newCP
}

func (cp *CP) initialiseDefaultConnector(connector int) {
	cp.connectors[connector] = &ConnectorInfo{
		availability:       core.AvailabilityTypeOperative,
		currentTransaction: 0,
		statusC:            make(chan struct{}),
		measurements:       make(map[string]types.SampledValue),
	}
	/*
		{
			1: ,
			2: {availability: core.AvailabilityTypeOperative, currentTransaction: 0, statusC: make(chan struct{}), measurements: make(map[string]types.SampledValue)},
		}
	*/
}

func (cp *CP) GetConnectorByID(ID int) *ConnectorInfo {
	return cp.getConnectorByID(ID)
}

func (cp *CP) getConnectorByID(ID int) *ConnectorInfo {
	// erstelle automatisch connectoren wenn eine unbekannte id abgefragt wird
	if ID == 0 {

		con, ok := cp.connectors[1]
		if !ok {
			cp.initialiseDefaultConnector(1)
			con = cp.connectors[1]
		}
		return con
	} else {
		con, ok := cp.connectors[ID]
		if !ok {
			cp.initialiseDefaultConnector(ID)
			con = cp.connectors[ID]
		}

		return con
	}
}

func (cp *CP) isValidConnectorID(ID int) bool {
	_, ok := cp.connectors[ID]
	return ok || ID == 0
}

func (cp *CP) TestClock(clock clock.Clock, connector int) {
	cp.clock = clock
}

func (cp *CP) ID() string {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.id
}

func (cp *CP) RegisterID(id string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.id != "" {
		panic("ocpp: cannot re-register id")
	}

	cp.id = id
}

/*
func (cp *CP) Connector() int {
	return cp.connector
}
*/

func (cp *CP) connect(connect bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.connected = connect

	if connect {
		cp.once.Do(func() {
			close(cp.connectC)
		})
	}
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

func (cp *CP) Initialized(connector int) error {
	// trigger status
	time.AfterFunc(cp.timeout/2, func() {
		select {
		case <-cp.getConnectorByID(connector).statusC:
			return
		default:
			Instance().TriggerMessageRequest(cp.ID(), core.StatusNotificationFeatureName, func(request *remotetrigger.TriggerMessageRequest) {
				request.ConnectorId = &connector
			})
		}
	})

	// wait for status
	select {
	case <-cp.getConnectorByID(connector).statusC:
		return nil
	case <-time.After(cp.timeout):
		return api.ErrTimeout
	}
}

// TransactionID returns the current transaction id
func (cp *CP) TransactionID(connector int) (int, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.connected {
		return 0, api.ErrTimeout
	}

	return cp.getConnectorByID(connector).txnId, nil
}

func (cp *CP) Status(connector int) (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if !cp.connected || !cp.isValidConnectorID(connector) {
		return res, api.ErrTimeout
	}

	if cp.getConnectorByID(connector).status.ErrorCode != core.NoError {
		return res, fmt.Errorf("%s: %s", cp.getConnectorByID(connector).status.ErrorCode, cp.getConnectorByID(connector).status.Info)
	}

	switch cp.getConnectorByID(connector).status.Status {
	case core.ChargePointStatusAvailable, // "Available"
		core.ChargePointStatusUnavailable: // "Unavailable"
		res = api.StatusA
	case
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		res = api.StatusB
	case core.ChargePointStatusCharging: // "Charging"
		res = api.StatusC
	case core.ChargePointStatusReserved, // "Reserved"
		core.ChargePointStatusFaulted: // "Faulted"
		return api.StatusF, fmt.Errorf("chargepoint status: %s", cp.getConnectorByID(connector).status.ErrorCode)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", cp.getConnectorByID(connector).status.Status)
	}

	return res, nil
}

// WatchDog triggers meter values messages if older than timeout.
// Must be wrapped in a goroutine.
func (cp *CP) WatchDog(timeout time.Duration, connector int) {
	for ; true; <-time.Tick(timeout) {
		cp.mu.Lock()
		update := cp.getConnectorByID(connector).txnId != 0 && cp.clock.Since(cp.getConnectorByID(connector).meterUpdated) > timeout
		cp.mu.Unlock()

		if update {
			Instance().TriggerMeterValuesRequest(cp.ID(), connector)
			cp.log.TRACE.Printf("TriggerMeterValuesRequest for Id and ConnectorID: %s , %d ", cp.id, connector)
		}
	}
}

func (cp *CP) isTimeout(connector int) bool {
	return cp.timeout > 0 && cp.isValidConnectorID(connector) && cp.clock.Since(cp.getConnectorByID(connector).meterUpdated) > cp.timeout
}

//var _ api.Meter = (*CP)(nil)

func (cp *CP) CurrentPower(connector int) (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.connected && cp.isValidConnectorID(connector) {
		return 0, api.ErrTimeout
	}

	// zero value on timeout when not charging
	if cp.isTimeout(connector) {
		if cp.getConnectorByID(connector).txnId != 0 {
			return 0, api.ErrTimeout
		}

		return 0, nil
	}

	if m, ok := cp.getConnectorByID(connector).measurements[string(types.MeasurandPowerActiveImport)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

//var _ api.MeterEnergy = (*CP)(nil)

func (cp *CP) TotalEnergy(connector int) (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.connected && !cp.isValidConnectorID(connector) {
		return 0, api.ErrTimeout
	}

	// fallthrough for last value on timeout when not charging
	if cp.getConnectorByID(connector).txnId != 0 && cp.isTimeout(connector) {
		return 0, api.ErrTimeout
	}

	if m, ok := cp.getConnectorByID(connector).measurements[string(types.MeasurandEnergyActiveImportRegister)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit) / 1e3, err
	}

	return 0, api.ErrNotAvailable
}

func scale(f float64, scale types.UnitOfMeasure) float64 {
	switch {
	case strings.HasPrefix(string(scale), "k"):
		return f * 1e3
	case strings.HasPrefix(string(scale), "m"):
		return f / 1e3
	default:
		return f
	}
}

func getKeyCurrentPhase(phase int) string {
	return string(types.MeasurandCurrentImport) + "@L" + strconv.Itoa(phase)
}

//var _ api.PhaseCurrents = (*CP)(nil)

func (cp *CP) Currents(connector int) (float64, float64, float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.connected && !cp.isValidConnectorID(connector) {
		return 0, 0, 0, api.ErrTimeout
	}

	// zero value on timeout when not charging
	if cp.isTimeout(connector) {
		if cp.getConnectorByID(connector).txnId != 0 {
			return 0, 0, 0, api.ErrTimeout
		}

		return 0, 0, 0, nil
	}

	currents := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := cp.getConnectorByID(connector).measurements[getKeyCurrentPhase(phase)]
		if !ok {
			return 0, 0, 0, api.ErrNotAvailable
		}

		f, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid current for phase %d: %w", phase, err)
		}

		currents = append(currents, scale(f, m.Unit))
	}

	return currents[0], currents[1], currents[2], nil
}
