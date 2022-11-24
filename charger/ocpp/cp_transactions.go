package ocpp

import (
	// "fmt"
	// "strconv"
	// "strings"
	"sync"
	// "time"

	// "github.com/evcc-io/evcc/api"
	// "github.com/evcc-io/evcc/util"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

type TransactionState int

const (
	TransactionUndefined TransactionState = iota
	TransactionStarting
	TransactionRunning
	TransactionSuspended
	TransactionStopping
	TransactionFinished
)

type Transaction struct {
	Id int
	status TransactionState
	ConnectorId int

	// state signalling
	statusMu sync.Mutex
	statusChanged chan struct{}
	// StartedC chan struct{}
	// StoppedC chan struct{}
}

func (tnx *Transaction) broadcastStatusChange() {
	if tnx.statusChanged != nil {
		close(tnx.statusChanged)
	}
	tnx.statusChanged = make(chan struct{})
}

func (tnx *Transaction) SetStatus(status TransactionState) {
	tnx.statusMu.Lock()
	defer tnx.statusMu.Unlock()

	// preveting double status update
	if tnx.status != status {
		tnx.status = status
		tnx.broadcastStatusChange()
	}
}

func (tnx *Transaction) Status() TransactionState {
	tnx.statusMu.Lock()
	defer tnx.statusMu.Unlock()
	return tnx.status
}

func (tnx *Transaction) HasStatusChanged() <-chan struct{} {
	return tnx.statusChanged
}

func NewTransaction(id int) *Transaction {
	return &Transaction {
		Id: id,
		ConnectorId: 1,
		status: TransactionUndefined,
		statusChanged: make(chan struct{}),
	}
}

