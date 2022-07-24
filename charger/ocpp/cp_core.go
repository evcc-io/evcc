package ocpp

import (
	"strconv"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	messageExpiry     = 30 * time.Second
	transactionExpiry = time.Hour
)

func (cp *CP) Authorize(request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	// TODO check if this authorizes foreign RFID tags
	res := &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cp *CP) BootNotification(request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request != nil {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		cp.boot = request
		cp.initialized.Broadcast()
	}

	res := &core.BootNotificationConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    60, // TODO
		Status:      core.RegistrationStatusAccepted,
	}

	return res, nil
}

// timestampValid returns false if status timestamps are outdated
func (cp *CP) timestampValid(t time.Time) bool {
	// reject if expired
	if time.Since(t) > messageExpiry {
		return false
	}

	// assume having a timestamp is better than not
	if cp.status.Timestamp == nil {
		return true
	}

	// reject older values than we already have
	return !t.Before(cp.status.Timestamp.Time)
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request != nil {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		if cp.status == nil {
			cp.status = request
			cp.initialized.Broadcast()
		} else if request.Timestamp == nil || cp.timestampValid(request.Timestamp.Time) {
			cp.status = request
		} else {
			cp.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, cp.status.Timestamp)
		}
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusRejected,
	}

	return res, nil
}

func (cp *CP) update() {
	cp.mu.Lock()
	cp.updated = time.Now()
	cp.mu.Unlock()
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	cp.update()
	res := &core.HeartbeatConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
	}

	if !cp.meterTickerRunning && cp.meterSupported {
		Instance().TriggerMeterValueRequest(cp)
	}

	return res, nil
}

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)
	if request.TransactionId != nil {
		cp.log.TRACE.Printf("TransactionId: %+v", *request.TransactionId)
	}

	if request != nil {
		cp.mu.Lock()
		cp.setMeterValues(request)

		if energy, ok := cp.measurements[string(types.MeasurandEnergyActiveImportRegister)]; ok {
			v, _ := strconv.ParseInt(energy.Value, 10, 64)
			cp.currentTransaction.Charged = v - cp.currentTransaction.MeterValueStart
		}

		cp.mu.Unlock()
	}

	return new(core.MeterValuesConfirmation), nil
}

func getSampleKey(s types.SampledValue) string {
	if s.Phase != "" {
		return string(s.Measurand) + "@" + string(s.Phase)
	}

	return string(s.Measurand)
}

func (cp *CP) setMeterValues(request *core.MeterValuesRequest) {
	for _, meterValue := range request.MeterValue {
		// ignore old meter value requests
		if meterValue.Timestamp.Time.After(cp.meterUpdated) {
			for _, sample := range meterValue.SampledValue {
				cp.measurements[getSampleKey(sample)] = sample
			}
		}
	}
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	// create new transaction
	if request != nil {
		if time.Since(request.Timestamp.Time) < transactionExpiry { // only respect transactions in the last hour
			cp.mu.Lock()
			cp.currentTransaction = NewTransaction(cp.currentTransaction.ID+1, request.IdTag, request.Timestamp.Time, request.MeterStart)

			cp.mu.Unlock()

			res.TransactionId = cp.currentTransaction.ID

			if cp.meterSupported && !cp.meterTickerRunning && time.Since(request.Timestamp.Time) < messageExpiry {
				go func() {
					cp.log.TRACE.Printf("starting meter value ticker")
					cp.meterTickerRunning = true
					cp.measureDoneCh = make(chan struct{})
					ticker := time.NewTicker(15 * time.Second)

					defer cp.log.TRACE.Printf("exiting meter value ticker")
					for {
						select {
						case <-ticker.C:
							Instance().TriggerMeterValueRequest(cp)
						case <-cp.measureDoneCh:
							cp.log.TRACE.Printf("returning from meter value requests")
							cp.meterTickerRunning = false
							return
						}
					}
				}()
			}
		} else {
			// TODO: Handle old transactions e.g. store them
			res.TransactionId = 1 // change 1 to the last known global transaction. Needs persistence
		}
	}

	return res, nil
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	// reset transaction
	if request != nil {
		cp.mu.Lock()

		cp.currentTransaction.Finish(request.IdTag, request.Timestamp.Time, request.MeterStop)

		cp.mu.Unlock()

		// TODO: Handle old transaction. Store them, check for the starting transaction event
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	if cp.meterSupported {
		if cp.meterTickerRunning {
			cp.measureDoneCh <- struct{}{}
		}

		Instance().TriggerMeterValueRequest(cp)
	}

	return res, nil
}

func (cp *CP) DiagnosticStatusNotification(request *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	return &firmware.DiagnosticsStatusNotificationConfirmation{}, nil
}

func (cp *CP) FirmwareStatusNotification(request *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}
