package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/security"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// Security event severity levels
const (
	SecuritySeverityCritical = "CRITICAL"
	SecuritySeverityHigh     = "HIGH"
	SecuritySeverityMedium   = "MEDIUM"
	SecuritySeverityLow      = "LOW"
)

// cp actions

func (cs *CS) OnAuthorize(id string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	// no cp handler

	res := &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cs *CS) OnBootNotification(id string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnBootNotification(request)
	}

	res := &core.BootNotificationConfirmation{
		CurrentTime: types.Now(),
		Interval:    int(Timeout.Seconds()),
		Status:      core.RegistrationStatusPending, // not accepted during startup
	}

	return res, nil
}

func (cs *CS) OnDataTransfer(id string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	// no cp handler

	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusAccepted,
	}

	return res, nil
}

func (cs *CS) OnHeartbeat(id string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	// no cp handler

	res := &core.HeartbeatConfirmation{
		CurrentTime: types.Now(),
	}

	return res, nil
}

func (cs *CS) OnMeterValues(id string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnMeterValues(request)
	}

	return new(core.MeterValuesConfirmation), nil
}

func (cs *CS) OnStatusNotification(id string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cs.mu.Lock()
	// cache status for future cp connection
	if reg, ok := cs.regs[id]; ok && request != nil {
		reg.mu.Lock()
		reg.status[request.ConnectorId] = request
		reg.mu.Unlock()
	}
	cs.mu.Unlock()

	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnStatusNotification(request)
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cs *CS) OnStartTransaction(id string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnStartTransaction(request)
	}

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cs *CS) OnStopTransaction(id string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		cp.OnStopTransaction(request)
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept old pending stop message during startup
		},
	}

	return res, nil
}

func (cs *CS) OnSecurityEventNotification(id string, request *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationResponse, error) {
	eventType := request.Type
	timestamp := request.Timestamp
	techInfo := request.TechInfo

	// Map event types to severity levels based on common OCPP security practices
	severity := getSecurityEventSeverity(eventType)

	// Log the security event with appropriate level
	switch severity {
	case SecuritySeverityCritical:
		cs.log.ERROR.Printf("charge point %s: %s security event %s at %s (tech: %s)",
			id, SecuritySeverityCritical, eventType, timestamp, techInfo)
	case SecuritySeverityHigh:
		cs.log.WARN.Printf("charge point %s: %s security event %s at %s (tech: %s)",
			id, SecuritySeverityHigh, eventType, timestamp, techInfo)
	case SecuritySeverityMedium:
		cs.log.WARN.Printf("charge point %s: %s security event %s at %s (tech: %s)",
			id, SecuritySeverityMedium, eventType, timestamp, techInfo)
	case SecuritySeverityLow:
		cs.log.INFO.Printf("charge point %s: %s security event %s at %s (tech: %s)",
			id, SecuritySeverityLow, eventType, timestamp, techInfo)
	default:
		cs.log.INFO.Printf("charge point %s: security event %s at %s (tech: %s)",
			id, eventType, timestamp, techInfo)
	}

	// Acknowledge the security event
	return &security.SecurityEventNotificationResponse{}, nil
}

// getSecurityEventSeverity maps security event types to severity levels
func getSecurityEventSeverity(eventType string) string {
	switch eventType {
	// Critical events that require immediate attention
	case "InvalidFirmwareSignature", "InvalidFirmwareSigningCertificate",
		 "InvalidCentralSystemCertificate", "InvalidChargePointCertificate",
		 "MemoryExhaustion":
		return SecuritySeverityCritical

	// High severity events indicating potential security issues
	case "FirmwareMismatch", "InvalidMessages", "SecurityLogWasCleared",
		 "ReconfigurationOfSecurityParameters":
		return SecuritySeverityHigh

	// Medium severity events for monitoring
	case "StartupOfTheDevice", "ResetOrReboot":
		return SecuritySeverityMedium

	// Low severity informational events
	case "SettingSystemTime":
		return SecuritySeverityLow

	// Default to critical for unknown event types
	default:
		return SecuritySeverityCritical
	}
}

// Security extension handlers for OCPP 1.6j

func (cs *CS) OnSignCertificate(id string, request *security.SignCertificateRequest) (*security.SignCertificateResponse, error) {
	cs.log.INFO.Printf("charge point %s: certificate signing request received", id)

	// For now, reject certificate signing requests as evcc doesn't implement PKI
	return &security.SignCertificateResponse{
		Status: types.GenericStatusRejected,
	}, nil
}

func (cs *CS) OnCertificateSigned(id string, request *security.CertificateSignedRequest) (*security.CertificateSignedResponse, error) {
	cs.log.INFO.Printf("charge point %s: signed certificate received", id)

	// Accept the certificate notification
	return &security.CertificateSignedResponse{
		Status: security.CertificateSignedStatusAccepted,
	}, nil
}
